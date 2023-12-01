package timestamp

import "time"

// T is a convenience type for UNIX 64 bit timestamps of 1 second
// precision.
type T int64

// Now returns the current UNIX timestamp of the current second.
func Now() T {
	return T(time.Now().Unix())
}

// Time converts a timestamp.Time value into a canonical UNIX 64 bit 1 second
// precision timestamp.
func (t T) Time() time.Time {
	return time.Unix(int64(t), 0)
}
