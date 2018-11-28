package main

import (
	"encoding/binary"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/nanorobocop/worldping/db"
	"github.com/nanorobocop/worldping/task"
	"github.com/shirou/gopsutil/load"

	ping "github.com/digineo/go-ping"
)

const (
	dbPublishSize = 10000

	grandMaxGoroutines = 10000
)

var dbAddr = os.Getenv("DB_ADDRESS")
var dbPort = os.Getenv("DB_PORT")
var dbUsername = os.Getenv("DB_USERNAME")
var dbPassword = os.Getenv("DB_PASSWORD")
var dbName = os.Getenv("DB_NAME")
var dbTable = os.Getenv("DB_TABLE")
var maxLoad, _ = strconv.ParseFloat(os.Getenv("MAX_LOAD"), 64)

type envStruct struct {
	dbConn     db.DB
	gracefulCh chan os.Signal
	wg         sync.WaitGroup
}

func (env *envStruct) initialize() {
	if err := env.dbConn.Open(); err != nil {
		log.Fatalf("[FATAL] Cannot open connection to database: %+v", err)
	}

	if err := env.dbConn.Ping(); err != nil {
		log.Fatalf("[FATAL] Cannot ping DB: %v", err)
	}

	log.Println("[INFO] Creating table if not exists..")
	if err := env.dbConn.CreateTable(); err != nil {
		log.Fatalf("[FATAL] Table creation failed: %v", err)
	}
	log.Println("[INFO] Table creation finished")

}

func (env *envStruct) getTasks(tasksCh chan task.Task) {
	curIP, err := env.dbConn.GetMaxIP()
	if err != nil {
		log.Printf("[INFO] Could not get curIP from db: %+v", err)
		log.Printf("[INFO] Using 0 for curIP")
	}

	curIP++
	for {
		log.Printf("[INFO] getTasks: Sending task with ip=%d", curIP)
		tasksCh <- task.Task{IP: curIP}
		curIP++
	}
}

func pingf(ip uint32, resultCh chan task.Task, guard chan struct{}) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, ip)

	log.Printf("[INFO] pingf: Pinging %v", ip)

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
	log.Printf("[INFO] pingf: %d : %v", ip, success)

	resultCh <- task.Task{IP: ip, Ping: success}
	<-guard
}

func (env *envStruct) schedule(taskCh, resultCh chan task.Task, loadCh chan float64) {

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
			log.Printf("[INFO] Goroutines: %v (%v)", len(guard), maxGoroutines)
			guard <- struct{}{}
			go func() {
				pingf(task.IP, resultCh, guard)
			}()
		default:
		}
	}
}

func (env *envStruct) sendStat(resultCh chan task.Task) {
	defer env.wg.Done()

	var results task.Tasks = make([]task.Task, 0, dbPublishSize)

	for {
		select {
		case result := <-resultCh:
			results = append(results, result)
			if len(results) == dbPublishSize {
				if err := env.dbConn.Save(results); err != nil {
					log.Printf("[ERROR] Problem at saving result to database: %s", err)
				}
				results = results[:0]
			}
		case <-env.gracefulCh:
			log.Printf("[INFO] Received signal for shutdown. Storing results to DB. Results: %+v", results)
			if err := env.dbConn.Save(results); err != nil {
				log.Printf("[ERROR] Problem at saving result to database: %s", err)
			}
			log.Printf("[INFO] Data successfully stored")
			return
		}
	}
}

func (env *envStruct) getLoad(loadCh chan float64) {
	for {
		avg, err := load.Avg()
		if err != nil {
			log.Printf("[ERROR] Unable to extract load average: %+v", err)
		}
		cores := runtime.NumCPU()
		load := avg.Load1 / float64(cores)
		loadCh <- load
		time.Sleep(time.Second)
	}
}

func main() {
	log.Println("[INFO] Worldping started")

	if maxLoad <= 0 || maxLoad > 100 {
		log.Fatalf("[FATAL] Wrong value maxLoad=%v (should be between 0 and 100)", maxLoad)
	}

	taskCh := make(chan task.Task)
	resultCh := make(chan task.Task)
	loadCh := make(chan float64)
	gracefulCh := make(chan os.Signal)

	signal.Notify(gracefulCh, syscall.SIGTERM, syscall.SIGINT)

	env := envStruct{
		dbConn: &db.Postgres{
			DBAddr:     dbAddr,
			DBPort:     dbPort,
			DBName:     dbName,
			DBTable:    dbTable,
			DBUsername: dbUsername,
			DBPassword: dbPassword,
		},
		gracefulCh: gracefulCh,
	}

	env.initialize()

	defer env.dbConn.Close()

	go env.getLoad(loadCh)

	go env.getTasks(taskCh)

	go env.schedule(taskCh, resultCh, loadCh)

	env.wg.Add(1)
	go env.sendStat(resultCh)

	env.wg.Wait()

	log.Println("[INFO] Application stopped")
}
