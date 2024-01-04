package pools

import (
	"sync"
	"unsafe"
)

const MAX_LOCKS = 50

var namedMutexPool = make([]sync.Mutex, MAX_LOCKS)

//go:noescape
//go:linkname memhash runtime.memhash
func memhash(p unsafe.Pointer, h, s uintptr) uintptr

func NamedLock(name string) (unlock func()) {
	sptr := unsafe.StringData(name)
	idx := uint64(memhash(unsafe.Pointer(sptr), 0, uintptr(len(name)))) % MAX_LOCKS
	namedMutexPool[idx].Lock()
	return namedMutexPool[idx].Unlock
}
