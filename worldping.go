package main

import (
	"context"
	"encoding/binary"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/apsdehal/go-logger"
	"github.com/nanorobocop/worldping/db"
	"github.com/nanorobocop/worldping/pkg/types"
	"github.com/nanorobocop/worldping/pkg/utils"
	"github.com/shirou/gopsutil/load"

	ping "github.com/digineo/go-ping"

	_ "net/http/pprof"
)

// Pinger interface
type Pinger interface {
	Ping(*net.IPAddr, time.Duration) (time.Duration, error)
	Close()
}

type pinger struct{}

func (p *pinger) Ping(destination *net.IPAddr, timeout time.Duration) (time.Duration, error) {
	return p.Ping(destination, timeout)
}

func (p *pinger) Close() {
	p.Close()
}

const (
	dbPublishSize = 1<<15 - 1

	grandMaxGoroutines = 1000000
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

var dbAddr = os.Getenv("DB_ADDRESS")
var dbPort = os.Getenv("DB_PORT")
var dbUsername = os.Getenv("DB_USERNAME")
var dbPassword = os.Getenv("DB_PASSWORD")
var dbName = os.Getenv("DB_NAME")
var dbTable = os.Getenv("DB_TABLE")
var maxLoad, _ = strconv.ParseFloat(getEnv("MAX_LOAD", "1"), 64)
var l, _ = strconv.ParseInt(getEnv("LOG_LEVEL", "4"), 0, 0) // 4 - NOTICE, 5 - DEBUG
var logLevel = int(l)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

type envStruct struct {
	dbConn     db.DB
	ctx        context.Context
	gracefulCh chan os.Signal
	wg         sync.WaitGroup
	log        *logger.Logger
	pinger     Pinger
}

func (env *envStruct) initialize() {
	if err := env.dbConn.Open(); err != nil {
		env.log.Fatalf("Cannot open connection to database: %+v", err)
	}

	if err := env.dbConn.Ping(); err != nil {
		env.log.Fatalf("Cannot ping DB: %v", err)
	}

	env.log.Notice("Creating table if not exists..")
	if err := env.dbConn.CreateTable(); err != nil {
		env.log.Fatalf("Table creation failed: %v", err)
	}
	env.log.Notice("Table creation finished")
}

// getTasks requests DB for new range of IP addresses.
// One range is /8 subset = 2^24 = 16777216 addresses.
// Amount of ranges = 256.
// Each time range with oldest timestamp will be picked up.
func (env *envStruct) getTasks(tasksCh chan types.Task) {
	for {
		startIP, err := env.dbConn.GetOldestIP()
		if err != nil {
			env.log.Noticef("Could not get startIP from db (db empty?): %+v", err)
		}
		endIP := startIP + (1 << 24) - 1
		env.log.Noticef("Starting with range %s:%s (%d:%d)", utils.IPToStr(startIP), utils.IPToStr(endIP), startIP, endIP)

		for curIP := startIP; curIP <= endIP; curIP++ {
			select {
			case tasksCh <- types.Task{IP: curIP}:
				env.log.Debugf("getTasks: Sending task with ip=%d", curIP)
			case <-env.ctx.Done():
				return
			}
		}
	}
}

func (env *envStruct) pingf(ip uint32, resultCh chan types.Task, guard chan struct{}) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, ip)

	env.log.Debugf("pingf: Pinging %v", ip)

	_, err := env.pinger.Ping(&net.IPAddr{IP: buf}, 1*time.Second)
	var success bool

	if err == nil {
		success = true
	} else {
		success = false
	}
	env.log.Debugf("pingf: %d : %v", ip, success)

	resultCh <- types.Task{IP: ip, Ping: success}
	<-guard
}

func (env *envStruct) schedule(taskCh, resultCh chan types.Task, loadCh chan float64) {
	ticker := time.NewTicker(10 * time.Second)
	var curLoad float64
	var maxGoroutines = 1000
	guard := make(chan struct{}, grandMaxGoroutines)
	for {
		select {
		case curLoad = <-loadCh:
			if curLoad > maxLoad && maxGoroutines > 100 {
				maxGoroutines = maxGoroutines - 100
			} else {
				maxGoroutines = maxGoroutines + 100
			}
		case task := <-taskCh:
			for len(guard) > maxGoroutines {
				time.Sleep(time.Millisecond)
			}
			guard <- struct{}{}
			go env.pingf(task.IP, resultCh, guard)
		case <-ticker.C:
			env.log.Noticef("Goroutines: %v (%v)", len(guard), maxGoroutines)
		case <-env.ctx.Done():
			return
		}
	}
}

func (env *envStruct) sendStat(resultCh chan types.Task) {
	defer env.wg.Done()

	sendStatFunc := func(env *envStruct, results types.Tasks, guard chan struct{}) {
		pinged := 0
		var maxIP uint32
		for _, r := range results {
			if r.Ping == true {
				pinged++
			}
			if r.IP > maxIP {
				maxIP = r.IP
			}
		}
		env.log.Noticef("Saving results to DB: total %d, pinged %d, maxIP %v (%d)", len(results), pinged, utils.IPToStr(maxIP), maxIP)
		if err := env.dbConn.Save(results); err != nil {
			env.log.Errorf("Problem at saving result to database: %s", err)
		}
		<-guard
	}

	results := make([]types.Task, dbPublishSize)
	i := 0
	guard := make(chan struct{}, 20)

	for {
		select {
		case result := <-resultCh:
			results[i] = result
			i++
			if i == dbPublishSize {
				guard <- struct{}{}
				go sendStatFunc(env, results, guard)
				results = make([]types.Task, dbPublishSize)
				i = 0
				env.log.Noticef("DB Goroutines: %d", len(guard))
			}
		case <-env.ctx.Done():
			env.log.Noticef("Received signal for shutdown.")
			if len(results) != 0 {
				guard <- struct{}{}
				sendStatFunc(env, results[:i], guard)
			}
			return
		}
	}
}

func (env *envStruct) getLoad(loadCh chan float64) {
	ctxCh := env.ctx.Done()
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			avg, err := load.Avg()
			if err != nil {
				env.log.Errorf("Unable to extract load average: %+v", err)
			}
			cores := runtime.NumCPU()
			load := avg.Load1 / float64(cores)
			loadCh <- load
		case <-ctxCh:
			return
		}
	}
}

func main() {

	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	env := envStruct{
		dbConn: &db.Postgres{
			DBAddr:     dbAddr,
			DBPort:     dbPort,
			DBName:     dbName,
			DBTable:    dbTable,
			DBUsername: dbUsername,
			DBPassword: dbPassword,
		},
	}

	var err error

	env.log, err = logger.New("worldping", 0, os.Stdout)
	if err != nil {
		log.Fatal("[FATAL] Unable to initiate logger")
	}

	env.log.SetLogLevel(logger.LogLevel(logLevel))

	env.log.SetFormat("%{time} %{level} %{message}")

	env.log.Notice("Worldping started")

	if maxLoad <= 0 || maxLoad > 100 {
		env.log.Fatalf("Wrong value maxLoad=%v (should be between 0 and 100)", maxLoad)
	}

	taskCh := make(chan types.Task)
	resultCh := make(chan types.Task)
	loadCh := make(chan float64)
	env.gracefulCh = make(chan os.Signal)

	signal.Notify(env.gracefulCh, syscall.SIGTERM, syscall.SIGINT)

	var cancel context.CancelFunc
	env.ctx = context.Background()
	env.ctx, cancel = context.WithCancel(env.ctx)

	go func() {
		<-env.gracefulCh
		cancel()
	}()

	env.initialize()
	defer env.dbConn.Close()

	pinger, err := ping.New("0.0.0.0", "")
	if err != nil {
		env.log.Fatalf("Cannot initialize pinger: %v", err)
	}
	env.pinger = pinger
	defer env.pinger.Close()

	go env.getLoad(loadCh)

	go env.getTasks(taskCh)

	go env.schedule(taskCh, resultCh, loadCh)

	env.wg.Add(1)
	go env.sendStat(resultCh)

	env.wg.Wait()

	env.log.Notice("Application stopped")

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
}
