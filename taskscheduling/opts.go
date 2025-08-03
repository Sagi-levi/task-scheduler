package taskscheduling

// Opts is a function type used to modify or configure taskHandler instances
// prior to execution.
type Opts func(handler *taskHandler)

// WithRetry sets the retry count for a task by modifying its `taskHandler`
// configuration, if retries are less than 1, it defaults to 1 to ensure at least
// one execution attempt.
func WithRetry(retries int) Opts {
	if retries < 1 {
		retries = 1
	}
	return func(task *taskHandler) {
		task.retries = retries
	}
}

// WithName sets a custom name for a task by modifying its `taskHandler`
// configuration.
func WithName(name string) Opts {
	return func(task *taskHandler) {
		task.name = name
	}
}
