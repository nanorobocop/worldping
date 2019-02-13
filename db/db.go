package db

import (
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/nanorobocop/worldping/worldping"

	_ "github.com/lib/pq" // driver
)

// DB implements interface for database (Postgres initially)
type DB interface {
	Open() error
	Ping() error
	CreateTable() error
	GetMaxIP() (uint32, error)
	GetOldestIP() (uint32, error)
	Save(worldping.Tasks) error
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
	_, err = db.c.Query(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (ip int PRIMARY KEY, ping bool, timestamp timestamp);`, db.DBTable))
	return err
}

// DropTable drops table (for tests)
func (db *Postgres) DropTable() (err error) {
	_, err = db.c.Query(fmt.Sprintf(`DROP TABLE %s;`, db.DBTable))
	return err
}

// GetMaxIP return maximum IP in db
func (db *Postgres) GetMaxIP() (maxIP uint32, err error) {
	var signed int32
	err = db.c.QueryRow(fmt.Sprintf("SELECT MAX(ip) from %s;", db.DBTable)).Scan(&signed)
	return *worldping.IntToUint(signed), err
}

// GetOldestIP returns oldest IP from db
func (db *Postgres) GetOldestIP() (oldestIP uint32, err error) {
	var signed int32
	// SELECT range FROM generate_series(-2147483648, 2147483647, 16777216) AS range LEFT OUTER JOIN worldping on (range = ip) ORDER BY timestamp NULLS FIRST LIMIT 1;
	stmt := fmt.Sprintf("SELECT range FROM generate_series(%d, %d, %d) AS range LEFT OUTER JOIN %s on (range = ip) ORDER BY timestamp NULLS FIRST LIMIT 1;", math.MinInt32, math.MaxInt32, 1<<24, db.DBTable)
	err = db.c.QueryRow(stmt).Scan(&signed)
	return *worldping.IntToUint(signed), err
}

// Save commits information to db
func (db *Postgres) Save(results worldping.Tasks) (err error) {

	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*2)
	for i, result := range results {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, CURRENT_TIMESTAMP)", i*2+1, i*2+2)) // 0 -> ($1, $2), 1 -> ($3, $4), 2 -> ($5, $6)
		valueArgs = append(valueArgs, worldping.UintToInt(result.IP))
		valueArgs = append(valueArgs, result.Ping)
	}
	// worldping=> INSERT INTO worldping  (ip, ping) VALUES (1, false),(2,false) ON CONFLICT (ip) DO UPDATE SET ping = excluded.ping ;
	stmt := fmt.Sprintf("INSERT INTO %s (ip, ping, timestamp) VALUES %s ON CONFLICT (ip) DO UPDATE SET ping = excluded.ping, timestamp = CURRENT_TIMESTAMP", db.DBTable, strings.Join(valueStrings, ","))
	_, err = db.c.Exec(stmt, valueArgs...)
	return err
}

// Close closes connection to DB
func (db *Postgres) Close() error {
	return db.c.Close()
}
