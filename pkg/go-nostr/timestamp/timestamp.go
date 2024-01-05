package timestamp

import "time"

type T int64

func Now() T {
	return T(time.Now().Unix())
}

func (t T) Time() time.Time {
	return time.Unix(int64(t), 0)
}
