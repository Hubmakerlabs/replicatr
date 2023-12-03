package timestamp

import (
	"fmt"
	"time"
)

// T is a convenience type for UNIX 64 bit timestamps of 1 second
// precision.
type T int64

// Tp is a synonym that makes it possible to use this value with the extra
// feature of the property of set/unset by the Ptr function which takes the
// address.
//
// This is to
type Tp T

// Now returns the current UNIX timestamp of the current second.
func Now() T {
	return T(time.Now().Unix())
}

// Time converts a timestamp.Time value into a canonical UNIX 64 bit 1 second
// precision timestamp.
func (t T) Time() time.Time {
	return time.Unix(int64(t), 0)
}

// Ptr returns the pointer so values can register as nil and omitted.
func (t T) Ptr() *Tp {
	tp := Tp(t)
	return &tp
}

func (tp *Tp) T() T {
	t := T(*tp)
	return t
}

// FromTime returns a T from a time.Time
func FromTime(t time.Time) T { return T(t.Unix()) }

func (tp *Tp) String() string {
	return fmt.Sprint(tp.T())
}

func (tp *Tp) Clone() (tc *Tp) {
	cp := *tp
	return &cp
}
