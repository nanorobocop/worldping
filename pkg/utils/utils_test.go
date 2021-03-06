package utils

import "testing"

func TestIPToStr(t *testing.T) {
	tests := []struct {
		ipInt uint32
		ipStr string
	}{
		{
			ipInt: 0,
			ipStr: "0.0.0.0",
		},
		{
			ipInt: 1,
			ipStr: "0.0.0.1",
		},
		{
			ipInt: 1234567890, // 0x499602D2
			ipStr: "73.150.2.210",
		},
		{
			ipInt: 4294967295,
			ipStr: "255.255.255.255",
		},
	}

	for i, test := range tests {
		t.Logf("Test %d, %+v", i, test)
		actual := IPToStr(test.ipInt)
		if actual != test.ipStr {
			t.Errorf("FAILED: %s", actual)
		}
	}
}

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
		signed := UintToInt(test.unsigned)
		if *signed != test.signed {
			t.Errorf("Test %d FAILED: %d (actual) != %d (expected)", i, *signed, test.signed)
		}

		unsigned := IntToUint(test.signed)
		if *unsigned != test.unsigned {
			t.Errorf("Test %d FAILED: %d (actual) != %d (expected)", i, *unsigned, test.unsigned)
		}
	}
}
