package main

import (
	"errors"
	"log"
	"log/slog"
	"os"

	"task-scheduler/taskscheduling"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scheduler, err := taskscheduling.New(40, 4)
	if err != nil {
		log.Fatal(err)
	}
	task := func() error {
		return nil
	}
	failingTask := func() error {
		return errors.New("error")
	}
	err = scheduler.Register(task, taskscheduling.WithRetry(4), taskscheduling.WithName("number1"), taskscheduling.WithLogging(logger))
	if err != nil {
		log.Fatal(err)
	}
	err = scheduler.Register(task, taskscheduling.WithRetry(2), taskscheduling.WithName("number2"))
	if err != nil {
		log.Fatal(err)
	}
	err = scheduler.Register(task, taskscheduling.WithLogging(logger))
	if err != nil {
		log.Fatal(err)
	}
	err = scheduler.Register(failingTask, taskscheduling.WithRetry(4), taskscheduling.WithName("number4"), taskscheduling.WithLogging(logger))
	if err != nil {
		log.Fatal(err)
	}
	scheduler.Run()
	scheduler.Stop()
	scheduler.Summary()
}
