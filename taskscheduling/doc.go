/*
Package taskscheduling provides a concurrent task execution framework for Go applications.

This package enables you to schedule and execute tasks concurrently.

The guidelines of this package include:
- Tasks must be registered before their scheduler starts running.
- Each scheduler can be run only once, otherwise panics can occur.
- Stop function can be called only after Run function was called, otherwise
panics can occur.

Configurable option to repeat tasks as wished by the user,option to change the
task name, option to log info about the task using the opts pattern.

The package provides an option to print a summary of your task executions.
*/
package taskscheduling

import (
	_ "embed"
)

//go:embed doc.go
var doc string
