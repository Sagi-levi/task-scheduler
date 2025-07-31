package taskscheduling

import (
	"errors"
	"sync"
)

// Scheduler manages the scheduling, execution, and reporting of tasks with
// controlled concurrency.
type Scheduler struct {
	pending               chan taskHandler
	shutdown              chan struct{}
	tasksWg               sync.WaitGroup
	concurrentWorkerLimit int
}

// New creates and returns a Scheduler with the specified task channel size
// and concurrentWorkerLimit. Using taskChannelBuffer and concurrentWorkerLimit
// smaller than 1 will return an error.
func New(taskChannelBuffer, concurrentWorkerLimit int) (*Scheduler, error) {
	var err error
	if taskChannelBuffer <= 0 {
		err = errors.Join(err, errors.New("task channel buffer size must be greater than 0"))
	}
	if concurrentWorkerLimit <= 0 {
		err = errors.Join(err, errors.New("concurrent worker limit must be greater than 0"))
	}
	if err != nil {
		return nil, err
	}
	return &Scheduler{
		pending:               make(chan taskHandler, taskChannelBuffer),
		shutdown:              make(chan struct{}),
		concurrentWorkerLimit: concurrentWorkerLimit,
	}, nil
}
