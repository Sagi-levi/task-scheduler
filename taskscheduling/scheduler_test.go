package taskscheduling

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestScheduler(t *testing.T) {
	t.Run("testing invalid and valid New function call", func(t *testing.T) {
		_, err := New(-4, -6)
		if err == nil {
			t.Fatal("expected error, got none")
		} else {
			t.Log("got expected error")
		}
		_, err = New(10, 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
	t.Run("testing scheduler flow", func(t *testing.T) {
		t.Log("registering tasks to scheduler buffer to ensure registration is limited and function as expected")
		bufferSize := 4
		s, err := New(bufferSize, 2)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < bufferSize-1; i++ {
			err = s.Register(func() error { return nil })
			if err != nil {
				t.Fatal(err)
			}
		}
		err = s.Register(func() error {
			return errors.New("test error")
		})
		// Buffer is full now, the next registration will raise an error.
		err = s.Register(func() error { return nil })
		if err == nil {
			t.Fatal("expected channel is full, got none")
		} else {
			t.Log("channel is full as expected")
		}
		s.Run()
		expected := bufferSize
		got := s.registered
		// It's not allowed to use Register after Run, we get the register
		// counter correctly after Run function.
		if expected != got {
			t.Fatalf("expected %v, got %v", expected, got)
		}
		t.Log(`registered counter is correct`)
		s.Stop()
		panicTask := func() error {
			return nil
		}
		err = s.Register(panicTask)
		// Since stoping the scheduler close the channel, we can't write new tasks to the
		// task channel.
		if err == nil {
			t.Fatalf("expected error when registering after Stop(), got none")
		}
		t.Log("got expected error when registering after Stop()")
		failedTasks := s.runCounters.failed.Load()
		expectedFailedTasks := int32(1)
		if expected != got {
			t.Errorf("expected %v, got %v", expectedFailedTasks, failedTasks)
		} else {
			t.Log(`failed tasks counter is correct`)
		}
		doneTasks := s.runCounters.done.Load()
		expectedDoneTasks := int32(4)
		if expectedDoneTasks != doneTasks {
			t.Errorf("expected %v, got %v", expectedDoneTasks, doneTasks)
		} else {
			t.Log(`done tasks counter is correct`)
		}
		resultsCounter := len(s.resultHandler.all)
		expectedResultsCounter := 4
		if expectedResultsCounter != resultsCounter {
			t.Errorf("expected %v, got %v", expectedResultsCounter, resultsCounter)
		} else {
			t.Log(`all counter is correct`)
		}
		filterResults := make([]result, 0)
		for _, result := range s.resultHandler.all {
			if result.isOk {
				filterResults = append(filterResults, result)
			}
		}
		filterResultsCounter := len(filterResults)
		// Deeper check of the resultHandler mechanism, comparing the number of tasks that
		// didn't failed.
		expectedFilterResultsCounter := 3
		if filterResultsCounter != expectedFilterResultsCounter {
			t.Errorf("expected %v, got %v", expectedFilterResultsCounter, filterResultsCounter)
		} else {
			t.Log(`isOk all counter is correct`)
		}
	})
	// Testing with retry, with name and with logging
	tests := []struct {
		name              string
		retries           int
		totalDone         int
		totalFailed       int
		taskName          string
		shouldFindMessage string
		shouldFind        bool
	}{
		{"test failed task with retry and with logging", 3, 3, 3, "", "registered", true},
		{"test failed task with out retry ,with name and with logging", 1, 1, 1, "best task", "banana", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(10, 10)
			if err != nil {
				t.Fatal(err)
			}
			task := func() error {
				return errors.New("test error")
			}
			logger, buf := createTestLogger()
			if tt.taskName != "" {
				err = s.Register(task, WithName(tt.taskName), WithLogging(logger))
			} else {
				err = s.Register(task, WithRetry(tt.retries), WithLogging(logger))
				if err != nil {
					t.Fatal(err)
				}
			}
			s.Run()
			s.Stop()
			expected := tt.totalDone
			got := s.runCounters.done.Load()
			if int32(expected) != got {
				t.Errorf("expected %v, got %v", expected, got)
			}
			expected = tt.totalFailed
			got = s.runCounters.failed.Load()
			if int32(expected) != got {
				t.Errorf("expected %v, got %v", expected, got)
			}

			// Asserting with name capability.
			if tt.taskName != "" {
				if s.resultHandler.all[0].task.name != tt.taskName {
					t.Errorf("expected task with name %v, got %v", tt.taskName, s.resultHandler.all[0].task.name)
				} else {
					t.Log(`task name updated as expected`)
				}
			}

			// Assert with logging
			output := buf.String()
			hasMessage := strings.Contains(output, tt.shouldFindMessage)
			if tt.shouldFind && !hasMessage {
				t.Errorf("Expected to find '%s' in logs, got: %s",
					tt.shouldFindMessage, output)
			} else if !tt.shouldFind && hasMessage {
				t.Errorf("Did not expect to find '%s' in logs, got: %s",
					tt.shouldFindMessage, output)
			} else {
				t.Log(`expected log messages were captured`)
			}
		})
	}
}

func createTestLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	var logger *slog.Logger
	logger = slog.New(slog.NewTextHandler(buf, nil))
	return logger, buf
}

// Example demonstrates the creation, registration, execution, and summary of
// tasks using the Scheduler infrastructure. Dropping the time key provides us
// consistent output.
func ExampleScheduler() {
	s, err := New(10, 1)
	if err != nil {
		panic(err)
	}
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				// Drop the time field, necessary for consistent output.
				return slog.Attr{}
			}
			return a

		}})
	logger := slog.New(logHandler)

	task := func() error {
		return errors.New("error")
	}
	err = s.Register(task, WithName("best task ever"), WithLogging(logger), WithRetry(1))
	if err != nil {
		panic(err)
	}
	s.Run()
	s.Stop()
	s.Summary()
	//Output:
	//level=INFO msg=registered
	//level=INFO msg="running task: best task ever with worker: 0"
	//level=INFO msg="running task-best task ever attempt 1"
	//level=INFO msg="failed task-best task ever attempt 1"
	//level=INFO msg="finished task-best task ever attempt 1"
	//╔════════════════════════════════════════╗
	//║       Scheduler Execution Report       ║
	//╠════════════════════════════════════════╣
	//║ Registered tasks        :     1         ║
	//║ Total executions        :     1         ║
	//║ Successful executions   :     0         ║
	//║ Failed executions       :     1         ║
	//╠════════════════════════════════════════╣
	//║            Per-Task Results            ║
	//╠════════════════════════════════════════╣
	//║ best task ever rep:  0 ➜ ERR            ║
	//╚════════════════════════════════════════╝
}
