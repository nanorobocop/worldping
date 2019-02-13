package worldping

import (
	"fmt"
	"unsafe"
)

// Task contains info about a task
type Task struct {
	IP   uint32
	Ping bool
}

// Tasks is an array of tasks
type Tasks []Task

// IPToStr converts
func IPToStr(ipInt uint32) string {
	octet0 := ipInt >> 24
	octet1 := ipInt << 8 >> 24
	octet2 := ipInt << 16 >> 24
	octet3 := ipInt << 24 >> 24
	return fmt.Sprintf("%d.%d.%d.%d", octet0, octet1, octet2, octet3)
}

// UintToInt converts uint to int IP representation
func UintToInt(u uint32) *int32 {
	i := (*int32)(unsafe.Pointer(&u))
	return i
}

// IntToUint converts int to uint IP representation
func IntToUint(u int32) *uint32 {
	i := (*uint32)(unsafe.Pointer(&u))
	return i
}
