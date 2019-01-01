package task

import "fmt"

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
