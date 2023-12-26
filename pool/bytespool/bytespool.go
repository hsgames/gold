package bytespool

import (
	"fmt"
	"sync"
)

var pools = [4]*pool{
	{min: 1, max: 4096, step: 512},           // 8 pools
	{min: 4097, max: 40960, step: 4096},      // 9 pools
	{min: 40961, max: 417792, step: 16384},   // 23 pools
	{min: 417793, max: 1925120, step: 65536}, // 23 pools
}

func init() {
	for _, v := range pools {
		v.init()
	}
}

type pool struct {
	min, max, step int
	pool           []sync.Pool
}

func (p *pool) init() {
	poolSize := (p.max - p.min + 1) / p.step
	p.pool = make([]sync.Pool, poolSize)

	for i := 0; i < poolSize; i++ {
		bytesSize := (p.min - 1) + (i+1)*p.step
		p.pool[i] = sync.Pool{New: func() any {
			return make([]byte, bytesSize)
		}}
	}
}

func (p *pool) pos(size int) int {
	if size < p.min {
		panic(fmt.Sprintf("bytespool: pos size [%d] < min [%d]", size, p.min))
	}

	idx := (size - p.min) / p.step

	if idx >= len(p.pool) {
		panic(fmt.Sprintf("bytespool: pos size [%d] out of range", p.max))
	}

	return idx
}

func (p *pool) get(size int) []byte {
	return p.pool[p.pos(size)].Get().([]byte)[:size]
}

func (p *pool) put(b []byte) {
	p.pool[p.pos(cap(b))].Put(b)
}

func Get(size int) []byte {
	for _, v := range pools {
		if size <= v.max {
			return v.get(size)
		}
	}

	return make([]byte, size)
}

func Put(b []byte) {
	if cap(b) == 0 {
		return
	}

	for _, v := range pools {
		if cap(b) <= v.max {
			v.put(b)
			return
		}
	}
}
