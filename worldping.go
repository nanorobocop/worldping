package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	ping "github.com/digineo/go-ping"
	"github.com/lib/pq"
)

type taskStruct struct {
	ip   uint32
	ping bool
}

type tasksStruct []taskStruct

type taskGetter interface {
	getTasks()
}

const (
	dbAddr  = "postgres"
	dbPort  = 5432
	dbName  = "worldping"
	dbUser  = "worldping"
	dbTable = "worldping"

	// dbAddr  = "127.0.0.1"
	// dbPort  = 5432
	// dbName  = "postgres"
	// dbUser  = "postgres"
	// dbTable = "worldping"

	dbPublishSize = 10000

	maxGoroutines = 128
)

//2018/10/12 09:57:38 getTasks - ERROR while querying for max ip: sql: Scan error on column index 0, name "max": converting driver.Value type <nil> ("<nil>") to a uint32: invalid syntax

type envStruct struct {
	dbAddr     string
	dbPort     int
	dbName     string
	dbUser     string
	dbPassword string
	dbTable    string
	dbConn     *sql.DB
}

func (env *envStruct) initialize() {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", env.dbAddr, env.dbPort, env.dbUser, env.dbPassword, env.dbName)
	var err error
	env.dbConn, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	if err = env.dbConn.Ping(); err != nil {
		log.Fatalf("getTasks - DB not pinged: %s", err)
	}

	log.Println("getTasks - Creating table if not exists..")
	_, err = env.dbConn.Query(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		ip int PRIMARY KEY,
		ping bool
	);
	`, env.dbTable))
	if err != nil {
		log.Fatalf("getTasks - Creating table failed: %s", err)
	}
	log.Println("getTasks - Creating table finished")

}

func (env *envStruct) getTasks(tasksCh chan taskStruct) {

	var maxIP uint32
	log.Println("getTasks - Requesting records..")
	sqlStatement := fmt.Sprintf("SELECT MAX(ip) from %s;", env.dbTable)
	row := env.dbConn.QueryRow(sqlStatement)
	switch err := row.Scan(&maxIP); err {
	case sql.ErrNoRows:
		log.Println("getTasks - No rows were, assigning 0")
		maxIP = 0
	case nil:
		log.Printf("getTasks - maxIP: %d", maxIP)
	default:
		log.Printf("getTasks - ERROR while querying for max ip: %s", err)
	}

	var curIP uint32
	curIP = maxIP + 1
	for {
		log.Printf("getTasks - Sending task with ip=%d", curIP)
		tasksCh <- taskStruct{ip: curIP}
		curIP++
	}

}

func pingf(ip uint32, resultCh chan taskStruct, guard chan struct{}) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, ip)

	log.Printf("pingf - Pinging %v", ip)

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
	log.Printf("pingf - %d : %v", ip, success)

	resultCh <- taskStruct{ip: ip, ping: success}
	<-guard
}

func schedule(taskCh, resultCh chan taskStruct) {
	guard := make(chan struct{}, maxGoroutines)
	for {
		select {
		case task := <-taskCh:
			guard <- struct{}{}
			go pingf(task.ip, resultCh, guard)
		}
	}

}

func aggregate(resultCh chan taskStruct, statCh chan tasksStruct) {
	var results tasksStruct = make([]taskStruct, 0, dbPublishSize)
	var finalResults tasksStruct = make([]taskStruct, dbPublishSize, dbPublishSize)
	for {
		select {
		case result := <-resultCh:
			results = append(results, result)
			if len(results) == dbPublishSize {
				log.Printf("aggregate - Sending aggregated results: %v", results)
				copy(finalResults, results)
				statCh <- finalResults
				results = results[:0]
			}
		}
	}
}

func (env *envStruct) sendStat(statCh chan tasksStruct) {
	for {
		select {
		case results := <-statCh:

			log.Printf("sendStat - START: %v", results)
			txn, err := env.dbConn.Begin()
			if err != nil {
				log.Println("begin")
				log.Fatal(err)
			}

			stmt, err := txn.Prepare(pq.CopyIn(env.dbTable, "ip", "ping"))
			if err != nil {
				log.Println("copyin")
				log.Fatal(err)
			}

			for _, result := range results {
				_, err = stmt.Exec(result.ip, result.ping)
				// log.Printf("res: %v", res)

				if err != nil {
					log.Fatal(err)
				}
			}

			_, err = stmt.Exec()
			if err != nil {
				log.Printf("%+v", err)
				log.Printf("%#+v", err)
				log.Fatal(err)
			}
			// log.Printf("res: %v", res)

			err = stmt.Close()
			if err != nil {
				log.Fatal(err)
			}

			err = txn.Commit()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("sendStat - END")
		}
	}
}

func main() {
	log.Println("Worldping started")

	taskCh := make(chan taskStruct)
	resultCh := make(chan taskStruct)
	statCh := make(chan tasksStruct)

	env := envStruct{
		dbAddr:     dbAddr,
		dbPort:     dbPort,
		dbName:     dbName,
		dbUser:     dbUser,
		dbPassword: os.Getenv("DB_PASSWORD"),
		dbTable:    dbTable,
	}

	p, err := ping.New("0.0.0.0", "")
	if err != nil {
		panic(err)
	}
	pinger := p

	_, err = pinger.Ping(&net.IPAddr{IP: []byte{2, 2, 2, 2}}, 10*time.Second)

	env.initialize()

	go env.getTasks(taskCh)

	go schedule(taskCh, resultCh)

	go aggregate(resultCh, statCh)

	go env.sendStat(statCh)

	var forever chan int
	forever <- 0

}
