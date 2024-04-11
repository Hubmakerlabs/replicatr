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
