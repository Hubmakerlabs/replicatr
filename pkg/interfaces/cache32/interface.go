package cache32

import "time"

type I[V any] interface {
	Get(k string) (v V, ok bool)
	Delete(k string)
	Set(k string, v V) bool
	SetWithTTL(k string, v V, d time.Duration) bool
}
