package taskscheduling

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// Scheduler manages the scheduling, execution, and reporting of tasks with
// controlled concurrency.
type Scheduler struct {
	pending               chan taskHandler
	shutdown              chan struct{}
	resultHandler         resultHandler
	tasksWg               sync.WaitGroup
	concurrentWorkerLimit int
	registered            int
	runCounters           runCounters
	inProgress            atomic.Bool
}

type resultHandler struct {
	buffer chan result
	all    []result
	lock   sync.Mutex
	wg     sync.WaitGroup
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
		resultHandler: resultHandler{
			buffer: make(chan result, concurrentWorkerLimit),
			lock:   sync.Mutex{},
			wg:     sync.WaitGroup{},
		},
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
		newTask.log("registered")
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
	s.resultHandler.wg.Add(1)
	go s.resultHandling()
	// We can measure the registered count using len of task channel because we won't
	// allow tasks to be registered after Run.
	s.registered = len(s.pending)
	for i := 0; i < s.concurrentWorkerLimit; i++ {
		s.tasksWg.Add(1)
		go func() {
			defer s.tasksWg.Done()
			for task := range s.pending {
				task.log("running task: %v with worker: %v", task.name, i)
				s.taskRetry(task)
			}
		}()
	}
	go s.runCleanup()
}

func (s *Scheduler) taskRetry(task taskHandler) {
	for i := 0; i < task.retries; i++ {
		task.log("running task-%v attempt %v", task.name, i+1)
		result := result{
			task: task,
			rep:  i,
		}
		if err := task.run(); err != nil {
			s.runCounters.failed.Add(1)
			task.log("failed task-%v attempt %v", task.name, i+1)
		} else {
			result.isOk = true
		}
		task.log("finished task-%v attempt %v", task.name, i+1)
		s.runCounters.done.Add(1)
		s.resultHandler.buffer <- result
		if result.isOk {
			break
		}
	}
}

func (s *Scheduler) resultHandling() {
	defer s.resultHandler.wg.Done()
	for res := range s.resultHandler.buffer {
		s.resultHandler.lock.Lock()
		s.resultHandler.all = append(s.resultHandler.all, res)
		s.resultHandler.lock.Unlock()
	}
}

func (s *Scheduler) getResults() []result {
	s.resultHandler.lock.Lock()
	defer s.resultHandler.lock.Unlock()
	results := make([]result, len(s.resultHandler.all))
	copy(results, s.resultHandler.all)
	return results
}

// Stop the execution of the Scheduler, stopping any ongoing tasks or processes.
// Stop can't be called more than once.
func (s *Scheduler) Stop() {
	s.inProgress.Store(false)
	close(s.pending)
	<-s.shutdown
}

func (s *Scheduler) runCleanup() {
	s.tasksWg.Wait()
	close(s.resultHandler.buffer)
	s.resultHandler.wg.Wait()
	close(s.shutdown)
}

// Summary generates and displays a detailed execution report, including statistics and per-task results.
func (s *Scheduler) Summary() {
	failed := s.runCounters.failed.Load()
	repetitions := s.runCounters.done.Load()
	succeeded := int(repetitions) - int(failed)

	results := s.getResults()

	longestName := 0
	for _, r := range results {
		if n := len(r.task.name); n > longestName {
			longestName = n
		}
	}

	const fixedPart = " rep:000 ➜ ERR"
	innerWidth := longestName + len(fixedPart)
	if innerWidth < 40 {
		innerWidth = 40
	}

	borderTop := "╔" + strings.Repeat("═", innerWidth) + "╗"
	borderMiddle := "╠" + strings.Repeat("═", innerWidth) + "╣"
	borderBottom := "╚" + strings.Repeat("═", innerWidth) + "╝"

	center := func(s string) string {
		if len(s) >= innerWidth {
			return s
		}
		left := (innerWidth - len(s)) / 2
		right := innerWidth - len(s) - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	}

	fmt.Println(borderTop)
	fmt.Printf("║%s║\n", center("Scheduler Execution Report"))
	fmt.Println(borderMiddle)

	statLine := func(label string, val int32) {
		txt := fmt.Sprintf("%-23s : %5d", label, val)
		fmt.Printf("║ %-*s ║\n", innerWidth-1, txt)
	}

	statLine("Registered tasks", int32(s.registered))
	statLine("Total executions", repetitions)
	statLine("Successful executions", int32(succeeded))
	statLine("Failed executions", failed)

	fmt.Println(borderMiddle)
	fmt.Printf("║%s║\n", center("Per-Task Results"))
	fmt.Println(borderMiddle)

	for _, r := range results {
		status := "OK "
		if !r.isOk {
			status = "ERR"
		}
		line := fmt.Sprintf("%-*s rep:%3d ➜ %s", longestName, r.task.name, r.rep, status)
		fmt.Printf("║ %-*s ║\n", innerWidth-1, line)
	}

	fmt.Println(borderBottom)
}
