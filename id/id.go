package id

import (
	"sync/atomic"
)

type Tmp struct {
	id uint64
}

func (t *Tmp) Next() uint64 {
	id := atomic.LoadUint64(&t.id)
	nid := atomic.AddUint64(&t.id, 1)

	if nid < id {
		panic("id: tmp id is overflow")
	}

	return nid
}
