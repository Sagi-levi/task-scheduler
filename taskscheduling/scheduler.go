package taskscheduling

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// Scheduler manages the scheduling, execution, and reporting of tasks with
// controlled concurrency.
type Scheduler struct {
	pending               chan taskHandler
	shutdown              chan struct{}
	tasksWg               sync.WaitGroup
	concurrentWorkerLimit int
	registered            int
	runCounters           runCounters
	inProgress            atomic.Bool
}

type runCounters struct {
	done   atomic.Int32
	failed atomic.Int32
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
	scheduler := &Scheduler{
		pending:               make(chan taskHandler, taskChannelBuffer),
		shutdown:              make(chan struct{}),
		concurrentWorkerLimit: concurrentWorkerLimit,
	}
	scheduler.inProgress.Store(true)
	return scheduler, nil

}

// Register adds a task or process to the Scheduler for execution as per its
// scheduling configuration.
func (s *Scheduler) Register(task Task, opts ...Opts) error {
	if !s.inProgress.Load() {
		return errors.New("scheduler is stopped")
	}
	taskId := uuid.New()
	newTask := taskHandler{
		fn:      task,
		id:      taskId,
		name:    fmt.Sprintf("task-%v", taskId),
		retries: 1,
	}
	for _, opt := range opts {
		opt(&newTask)
	}
	select {
	case s.pending <- newTask:
		return nil
	case <-s.shutdown:
		return errors.New("scheduler is stopped")
	default:
		return errors.New("channel is full")
	}
}

// Run initializes the Scheduler, beginning the execution of scheduled tasks or
// processes. Tasks aren't allowed to register after calling Run, and run can't
// be called more than once.
func (s *Scheduler) Run() {
	// We can measure the registered count using len of task channel because we won't
	// allow tasks to be registered after Run.
	s.registered = len(s.pending)
	for i := 0; i < s.concurrentWorkerLimit; i++ {
		s.tasksWg.Add(1)
		go func() {
			defer s.tasksWg.Done()
			for task := range s.pending {
				s.taskRetry(task)
			}
		}()
	}
	go s.runCleanup()
}

func (s *Scheduler) taskRetry(task taskHandler) {
	for i := 0; i < task.retries; i++ {
		err := task.run()
		s.runCounters.done.Add(1)
		if err != nil {
			s.runCounters.failed.Add(1)
		}
	}
}

func (s *Scheduler) runCleanup() {
	s.tasksWg.Wait()
	close(s.shutdown)
}

// Stop the execution of the Scheduler, stopping any ongoing tasks or processes.
// Stop can't be called more than once.
func (s *Scheduler) Stop() {
	s.inProgress.Store(false)
	close(s.pending)
	<-s.shutdown
}
