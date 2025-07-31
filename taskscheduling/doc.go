/*
Package taskscheduling provides a concurrent task execution framework for Go applications.

This package enables you to schedule and execute tasks concurrently.
*/
package taskscheduling

import (
	_ "embed"
)

//go:embed doc.go
var doc string
