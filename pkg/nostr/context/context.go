// Package context is a set of shorter names for the very stuttery context
// library.
package context

import (
	"context"
)

type (
	T = context.Context
	F = context.CancelFunc
	C = context.CancelCauseFunc
)

var (
	Bg          = context.Background
	Cancel      = context.WithCancel
	Timeout     = context.WithTimeout
	TODO        = context.TODO
	Value       = context.WithValue
	CancelCause = context.WithCancelCause
	Canceled    = context.Canceled
)
