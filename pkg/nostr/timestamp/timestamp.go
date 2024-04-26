package timestamp

import (
	"encoding/binary"
	"fmt"
	"time"
)

// T is a convenience type for UNIX 64 bit timestamps of 1 second
// precision.
type T int64

// Tp is a synonym that makes it possible to use this value with the extra
// feature of the property of set/unset by the Ptr function which takes the
// address.
type Tp T

// Now returns the current UNIX timestamp of the current second.
func Now() T { return T(time.Now().Unix()) }

// U64 returns the current UNIX timestamp of the current second as uint64.
func (t T) U64() uint64 { return uint64(t) }

// I64 returns the current UNIX timestamp of the current second as int64.
func (t T) I64() int64 { return int64(t) }

// Time converts a timestamp.Time value into a canonical UNIX 64 bit 1 second
// precision timestamp.
func (t T) Time() time.Time { return time.Unix(int64(t), 0) }

// Int returns the timestamp as an int.
func (t T) Int() int { return int(t) }

func (t T) Bytes() (b []byte) {
	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(t))
	return
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

// FromUnix converts from a standard int64 unix timestamp.
func FromUnix(t int64) T { return T(t) }

// FromBytes converts from a string of raw bytes.
func FromBytes(b []byte) T { return T(binary.BigEndian.Uint64(b)) }

func (tp *Tp) String() string {
	return fmt.Sprint(tp.T())
}

func (tp *Tp) Clone() (tc *Tp) {
	cp := *tp
	return &cp
}
