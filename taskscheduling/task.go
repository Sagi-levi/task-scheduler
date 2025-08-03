package taskscheduling

import (
	"github.com/google/uuid"
)

// Task defines a function type that performs a specific task or action and
// returns an error if it fails.
type Task func() error

type taskHandler struct {
	id      uuid.UUID
	name    string
	fn      Task
	retries int
}

func (t *taskHandler) run() error {
	return t.fn()
}

type result struct {
	task taskHandler
	rep  int
	isOk bool
}
