package main

import (
	"encoding/binary"
	"log"
	"net"
	"os"
	"runtime"
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

type envStruct struct {
	dbConn db.DB
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

func schedule(taskCh, resultCh chan task.Task, loadCh chan float64) {
	var curLoad float64
	var maxGoroutines int
	guard := make(chan struct{}, grandMaxGoroutines)
	for {
		select {
		case curLoad = <-loadCh:
			if curLoad > 1 && maxGoroutines > 10 {
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
			go pingf(task.IP, resultCh, guard)
		default:
		}
	}
}

func aggregate(resultCh chan task.Task, statCh chan task.Tasks) {
	var results task.Tasks = make([]task.Task, 0, dbPublishSize)
	var finalResults task.Tasks = make([]task.Task, dbPublishSize, dbPublishSize)
	for {
		select {
		case result := <-resultCh:
			results = append(results, result)
			if len(results) == dbPublishSize {
				log.Printf("[INFO] Aggregate: Sending aggregated results: %v", results)
				copy(finalResults, results)
				statCh <- finalResults
				results = results[:0]
			}
		}
	}
}

func (env *envStruct) sendStat(statCh chan task.Tasks) {
	for {
		select {
		case results := <-statCh:
			if err := env.dbConn.Save(results); err != nil {
				log.Printf("[ERROR] Problem at saving result to database: %s", err)
			}
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

	taskCh := make(chan task.Task)
	resultCh := make(chan task.Task)
	statCh := make(chan task.Tasks)
	loadCh := make(chan float64)

	env := envStruct{dbConn: &db.Postgres{
		DBAddr:     dbAddr,
		DBPort:     dbPort,
		DBName:     dbName,
		DBTable:    dbTable,
		DBUsername: dbUsername,
		DBPassword: dbPassword,
	}}

	env.initialize()

	go env.getLoad(loadCh)

	go env.getTasks(taskCh)

	go schedule(taskCh, resultCh, loadCh)

	go aggregate(resultCh, statCh)

	go env.sendStat(statCh)

	var forever chan int
	forever <- 0

}
