package taskscheduling

import (
	"fmt"
	"log/slog"

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
	logger  *slog.Logger
}

func (t *taskHandler) run() error {
	return t.fn()
}

func (t *taskHandler) log(format string, a ...any) {
	if t.logger != nil {
		msg := fmt.Sprintf(format, a...)
		t.logger.Info(msg)
	}
}

type result struct {
	task taskHandler
	rep  int
	isOk bool
}
