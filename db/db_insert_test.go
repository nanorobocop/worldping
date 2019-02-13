package db

import (
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/nanorobocop/worldping/worldping"
)

func TestDBInsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	db := Postgres{
		DBAddr:     "127.0.0.1",
		DBPort:     "5432",
		DBName:     "postgres",
		DBTable:    "testdb_hugeinsert",
		DBUsername: "postgres",
		DBPassword: "123456",
	}

	t.Logf("Preparing results")
	results := make([]worldping.Task, 1<<15-1)
	for i := range results {
		results[i] = worldping.Task{IP: uint32(i)}
	}

	db.Open()
	db.Ping()
	db.CreateTable()

	t.Logf("Preparing stmt")
	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*2)
	for i, result := range results {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, CURRENT_TIMESTAMP)", i*2+1, i*2+2)) // 0 -> ($1, $2), 1 -> ($3, $4), 2 -> ($5, $6)
		valueArgs = append(valueArgs, worldping.UintToInt(result.IP))
		valueArgs = append(valueArgs, result.Ping)
	}
	// worldping=> INSERT INTO worldping  (ip, ping) VALUES (1, false),(2,false) ON CONFLICT (ip) DO UPDATE SET ping = excluded.ping ;
	stmt := fmt.Sprintf("INSERT INTO %s (ip, ping, timestamp) VALUES %s ON CONFLICT (ip) DO UPDATE SET ping = excluded.ping, timestamp = CURRENT_TIMESTAMP", db.DBTable, strings.Join(valueStrings, ","))

	t.Logf("Executing stmt")
	start := time.Now()
	_, err := db.c.Exec(stmt, valueArgs...)
	duration := time.Since(start)
	if err != nil {
		t.Errorf("FAILED: %+v", err)
	}
	fmt.Printf("Duration: %v", duration)
	t.Logf("Dropping table")
	db.DropTable()
	t.Logf("Closing connection")
	db.Close()
}

func TestDBBulkInsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	db := Postgres{
		DBAddr:     "127.0.0.1",
		DBPort:     "5432",
		DBName:     "postgres",
		DBTable:    "testdb_bulkinsert",
		DBUsername: "postgres",
		DBPassword: "123456",
	}

	t.Logf("Preparing results")
	results := make([]worldping.Task, 1<<24)
	for i := range results {
		results[i] = worldping.Task{IP: uint32(i)}
	}

	db.Open()
	db.Ping()
	db.CreateTable()

	start := time.Now()
	t.Logf("Beginning txn")
	txn, err := db.c.Begin()
	if err != nil {
		log.Fatal(err)
	}

	t.Logf("CopyIn txn")
	stmt, err := txn.Prepare(pq.CopyIn(db.DBTable, "ip", "ping"))
	if err != nil {
		log.Fatal(err)
	}

	t.Logf("Exec for each IP")
	for _, result := range results {
		_, err = stmt.Exec(result.IP, result.Ping)
		if err != nil {
			log.Fatal(err)
		}
	}
	duration := time.Since(start)
	fmt.Printf("Duration: %v", duration)

	start = time.Now()
	t.Logf("Final exec")
	_, err = stmt.Exec()
	if err != nil {
		log.Fatal(err)
	}

	err = stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = txn.Commit()
	if err != nil {
		log.Fatal(err)
	}

	duration = time.Since(start)
	fmt.Printf("Duration: %v", duration)

	t.Logf("Dropping table")
	db.DropTable()
	t.Logf("Closing connection")
	db.Close()
}
