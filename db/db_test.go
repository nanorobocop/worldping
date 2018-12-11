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
			DBTable:    fmt.Sprintf("testdb_%d", rand.Intn(math.MaxInt16)),
			DBUsername: "postgres",
			DBPassword: "postgres",
		}
		db.Open()
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
