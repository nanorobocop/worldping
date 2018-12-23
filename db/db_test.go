package db

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/nanorobocop/worldping/task"
)

func TestIntUint(t *testing.T) {
	tests := []struct {
		unsigned uint32
		signed   int32
	}{
		{
			unsigned: 0,
			signed:   0,
		},
		{
			unsigned: 1,
			signed:   1,
		},
		{
			unsigned: 1<<31 - 1,
			signed:   1<<31 - 1,
		},
		{
			unsigned: 1 << 31,
			signed:   -1 << 31,
		},
		{
			unsigned: 1<<32 - 1,
			signed:   -1,
		},
	}

	for i, test := range tests {
		signed := uintToInt(test.unsigned)
		if *signed != test.signed {
			t.Errorf("Test %d FAILED: %d (actual) != %d (expected)", i, *signed, test.signed)
		}

		unsigned := intToUint(test.signed)
		if *unsigned != test.unsigned {
			t.Errorf("Test %d FAILED: %d (actual) != %d (expected)", i, *unsigned, test.unsigned)
		}
	}
}

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
			DBPassword: "postgres",
		}
		db.Open()
		db.Ping()
		db.CreateTable()
		results := task.Tasks{{IP: test}}
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
	resultsExceptLast := make([]task.Task, 255)
	for i := range resultsExceptLast {
		resultsExceptLast[i] = task.Task{IP: uint32(i * 1 << 24)}
	}

	steps := []struct {
		results  []task.Tasks
		expected uint32
	}{
		{
			results: []task.Tasks{
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
			DBPassword: "postgres",
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
