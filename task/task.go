package task

// Task contains info about a task
type Task struct {
	IP   uint32
	Ping bool
}

// Tasks is an array of tasks
type Tasks []Task
