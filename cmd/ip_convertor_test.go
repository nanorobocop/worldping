package main

import (
	"testing"
)

func TestParseInt(t *testing.T) {
	steps := []struct {
		str     string
		intVar  int32
		uintVar uint32
		strVar  string
		err     error
	}{
		{
			str:     "0",
			intVar:  int32(0),
			uintVar: uint32(0),
			strVar:  "0.0.0.0",
			err:     nil,
		},
		{
			str:     "1",
			intVar:  int32(1),
			uintVar: uint32(1),
			strVar:  "0.0.0.1",
			err:     nil,
		},
		{
			str:     "2147483647",
			intVar:  int32(2147483647),
			uintVar: uint32(2147483647),
			strVar:  "127.255.255.255",
			err:     nil,
		},
		{
			str:     "-2147483648",
			intVar:  int32(-2147483648),
			uintVar: uint32(2147483648),
			strVar:  "128.0.0.0",
			err:     nil,
		},
		{
			str:     "-1",
			intVar:  int32(-1),
			uintVar: uint32(4294967295),
			strVar:  "255.255.255.255",
			err:     nil,
		},
	}

	for i, step := range steps {
		uintVar, intVar, strVar, err := parseInt(&step.str)

		if (step.err == nil && (*intVar != step.intVar || *uintVar != step.uintVar || strVar != step.strVar)) ||
			(step.err != nil && err == nil) {
			t.Logf("Step %d FAILED:", i)
			t.Logf("Expected: (uint, int, str): %d, %d, %s", step.uintVar, step.intVar, step.strVar)
			t.Errorf("Actual: %d, %d, %s", *uintVar, *intVar, strVar)
		}
	}
}
