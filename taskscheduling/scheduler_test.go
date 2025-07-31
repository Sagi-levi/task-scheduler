package taskscheduling

import (
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
}
