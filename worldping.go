package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/apsdehal/go-logger"
	"github.com/nanorobocop/worldping/db"
	"github.com/nanorobocop/worldping/task"
	"github.com/shirou/gopsutil/load"

	ping "github.com/digineo/go-ping"
)

const (
	dbPublishSize = 10000

	grandMaxGoroutines = 10000
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

type envStruct struct {
	dbConn     db.DB
	ctx        context.Context
	gracefulCh chan os.Signal
	wg         sync.WaitGroup
	log        *logger.Logger
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

func (env *envStruct) getTasks(tasksCh chan task.Task) {
	curIP, err := env.dbConn.GetMaxIP()
	if err != nil {
		env.log.Noticef("Could not get curIP from db: %+v", err)
		env.log.Notice("Using 0 for curIP")
	}

	curIP++
	for {
		select {
		case tasksCh <- task.Task{IP: curIP}:
			env.log.Debugf("getTasks: Sending task with ip=%d", curIP)
			curIP++
		case <-env.ctx.Done():
			return
		}
	}
}

func (env *envStruct) pingf(ip uint32, resultCh chan task.Task, guard chan struct{}) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, ip)

	env.log.Debugf("pingf: Pinging %v", ip)

	p, err := ping.New("0.0.0.0", "")
	if err != nil {
		panic(err)
	}
	pinger := p
	defer pinger.Close()

	_, err = pinger.Ping(&net.IPAddr{IP: buf}, 1*time.Second)
	var success bool

	if err == nil {
		success = true
	} else {
		success = false
	}
	env.log.Debugf("pingf: %d : %v", ip, success)

	resultCh <- task.Task{IP: ip, Ping: success}
	<-guard
}

func (env *envStruct) schedule(taskCh, resultCh chan task.Task, loadCh chan float64) {
	ticker := time.NewTicker(10 * time.Second)
	var curLoad float64
	var maxGoroutines = 100
	guard := make(chan struct{}, grandMaxGoroutines)
	for {
		select {
		case curLoad = <-loadCh:
			if curLoad > maxLoad && maxGoroutines > 10 {
				maxGoroutines = maxGoroutines - 1
			} else {
				maxGoroutines = maxGoroutines + 1
			}
		case task := <-taskCh:
			for len(guard) > maxGoroutines {
				time.Sleep(time.Millisecond)
			}
			guard <- struct{}{}
			go func() {
				env.pingf(task.IP, resultCh, guard)
			}()
		case <-ticker.C:
			env.log.Noticef("Goroutines: %v (%v)", len(guard), maxGoroutines)
		default:
		}
	}
}

func ipToStr(ipInt uint32) string {
	octet0 := ipInt >> 24
	octet1 := ipInt << 8 >> 24
	octet2 := ipInt << 16 >> 24
	octet3 := ipInt << 24 >> 24
	return fmt.Sprintf("%d.%d.%d.%d", octet0, octet1, octet2, octet3)
}

func (env *envStruct) sendStat(resultCh chan task.Task) {
	defer env.wg.Done()

	var results task.Tasks = make([]task.Task, 0, dbPublishSize)

	sendStatFunc := func(env *envStruct, results task.Tasks) {
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
		env.log.Noticef("Saving results to DB: total %d, pinged %d, maxIP %v (%d)", len(results), pinged, ipToStr(maxIP), maxIP)
		if err := env.dbConn.Save(results); err != nil {
			env.log.Errorf("Problem at saving result to database: %s", err)
		}
	}

	for {
		select {
		case result := <-resultCh:
			results = append(results, result)
			if len(results) == dbPublishSize {
				sendStatFunc(env, results)
				results = results[:0]
			}
		case <-env.ctx.Done():
			env.log.Noticef("Received signal for shutdown.")
			sendStatFunc(env, results)
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

	taskCh := make(chan task.Task)
	resultCh := make(chan task.Task)
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

	go env.getLoad(loadCh)

	go env.getTasks(taskCh)

	go env.schedule(taskCh, resultCh, loadCh)

	env.wg.Add(1)
	go env.sendStat(resultCh)

	env.wg.Wait()

	env.log.Notice("Application stopped")
}
