/*
Package taskscheduling provides a concurrent task execution framework for Go applications.

This package enables you to schedule and execute tasks concurrently.

The guidelines of this package include:
- Tasks must be registered before their scheduler starts running.
*/
package taskscheduling

import (
	_ "embed"
)

//go:embed doc.go
var doc string
