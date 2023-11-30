package time

import "time"

// Stamp is a convenience type for UNIX 64 bit timestamps of 1 second
// precision.
type Stamp int64

// Now returns the current UNIX timestamp of the current second.
func Now() Stamp {
	return Stamp(time.Now().Unix())
}

// Time converts a time.Time value into a canonical UNIX 64 bit 1 second
// precision timestamp.
func (t Stamp) Time() time.Time {
	return time.Unix(int64(t), 0)
}
