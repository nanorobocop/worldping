package task

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
