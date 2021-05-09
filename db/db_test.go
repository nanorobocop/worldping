package db

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/nanorobocop/worldping/pkg/types"
)

func TestIntUintIntegrational(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tests := []uint32{
		0,
		1,
		1<<31 - 1,
		1 << 31,
		1<<32 - 1,
	}

	for i, test := range tests {
		db := Postgres{
			DBAddr:     "127.0.0.1",
			DBPort:     "5432",
			DBName:     "postgres",
			DBTable:    fmt.Sprintf("testdb_%d", rand.Intn(math.MaxInt32)),
			DBUsername: "postgres",
			DBPassword: "123456",
		}

		if err := db.Open(); err != nil {
			t.Fatalf("Cannot open DB: %+v", err)
		}

		if err := db.Ping(); err != nil {
			t.Fatalf("Cannot ping DB: %+v", err)
		}

		if err := db.CreateTable(); err != nil {
			t.Fatalf("Cannot create table: %+v", err)
		}

		results := types.Tasks{{IP: test}}
		db.Save(results)
		actual, _ := db.GetMaxIP()
		if actual != test {
			t.Errorf("Test %d FAILED (db %s): actual %d != expected %d", i, db.DBTable, actual, test)
		} else {
			db.DropTable()
		}
		db.Close()
	}
}

func TestGetOldestIntegrational(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	resultsExceptLast := make([]types.Task, 255)
	for i := range resultsExceptLast {
		resultsExceptLast[i] = types.Task{IP: uint32(i * 1 << 24)}
	}

	steps := []struct {
		results  []types.Tasks
		expected uint32
	}{
		{
			results: []types.Tasks{
				resultsExceptLast,
			},
			expected: uint32(1<<32 - 1<<24),
		},
	}

	for i, step := range steps {
		db := Postgres{
			DBAddr:     "127.0.0.1",
			DBPort:     "5432",
			DBName:     "postgres",
			DBTable:    fmt.Sprintf("testdb_%d", rand.Intn(math.MaxInt16)),
			DBUsername: "postgres",
			DBPassword: "123456",
		}
		db.Open()
		db.Ping()
		db.CreateTable()
		for _, r := range step.results {
			db.Save(r)
		}

		actual, _ := db.GetOldestIP()
		if actual != step.expected {
			t.Errorf("Test %d FAILED (db %s): actual %d != expected %d", i, db.DBTable, actual, step.expected)
		} else {
			db.DropTable()
		}
		db.Close()
	}
}
