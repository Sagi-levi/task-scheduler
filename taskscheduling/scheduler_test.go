package taskscheduling

import (
	"errors"
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
	})
}
