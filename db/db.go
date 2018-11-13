package db

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/nanorobocop/worldping/task"
)

// DB implements interface for database (Postgres initially)
type DB interface {
	Open() error
	Ping() error
	CreateTable() error
	GetMaxIP() (uint32, error)
	Save(task.Tasks) error
	Close() error
}

// Postgres contains connection to Postgres
type Postgres struct {
	c                                                       *sql.DB
	DBAddr, DBPort, DBName, DBUsername, DBPassword, DBTable string
}

// Open opens db connection
func (db *Postgres) Open() (err error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", db.DBAddr, db.DBPort, db.DBUsername, db.DBPassword, db.DBName)
	db.c, err = sql.Open("postgres", connStr)
	return err
}

// Ping checks connection to db
func (db *Postgres) Ping() error {
	return db.c.Ping()
}

// CreateTable creates table if not exists
func (db *Postgres) CreateTable() (err error) {
	_, err = db.c.Query(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (ip int PRIMARY KEY, ping bool);`, db.DBTable))
	return err
}

// GetMaxIP return maximum IP in db
func (db *Postgres) GetMaxIP() (maxIP uint32, err error) {
	err = db.c.QueryRow(fmt.Sprintf("SELECT MAX(ip) from %s;", db.DBTable)).Scan(&maxIP)
	return maxIP, err
}

// Save commits information to db
func (db *Postgres) Save(results task.Tasks) (err error) {
	txn, err := db.c.Begin()
	if err != nil {
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn(db.DBTable, "ip", "ping"))
	if err != nil {
		return err
	}

	for _, result := range results {
		_, err = stmt.Exec(result.IP, result.Ping)
		if err != nil {
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	err = txn.Commit()
	return err
}

// Close closes connection to DB
func (db *Postgres) Close() error {
	return db.c.Close()
}
