package wctest

import (
	"errors"
	"testing"
	"time"
)

func TestRunWaitDone(t *testing.T) {
	// Define some sample tasks
	task1 := func() error {
		time.Sleep(1 * time.Second)
		return nil
	}

	task2 := func() error {
		time.Sleep(2 * time.Second)
		return errors.New("task2 error")
	}

	task3 := func() error {
		time.Sleep(3 * time.Second)
		return nil
	}

	// Test case where all tasks complete successfully
	err := RunWaitDone(5*time.Second, task1, task3)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Test case where one task returns an error
	err = RunWaitDone(5*time.Second, task1, task2, task3)
	if err == nil {
		t.Errorf("expected an error, got nil")
	}
}
