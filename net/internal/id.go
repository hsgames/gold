package internal

import "sync/atomic"

var id uint64

func NextId() uint64 {
	return atomic.AddUint64(&id, 1)
}
