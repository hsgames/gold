package queue

import (
	"sync"
)

type MPSCQueue struct {
	in         []interface{}
	out        []interface{}
	mu         sync.Mutex
	cond       *sync.Cond
	maxSize    int
	shrinkSize int
	closed     bool
}

func NewMPSCQueue(maxSize, shrinkSize int) *MPSCQueue {
	q := &MPSCQueue{
		maxSize:    maxSize,
		shrinkSize: shrinkSize,
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *MPSCQueue) Push(data interface{}) {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	if q.maxSize > 0 && len(q.in) >= q.maxSize {
		data = nil
	}
	q.in = append(q.in, data)
	q.closed = data == nil
	q.mu.Unlock()
	q.cond.Signal()
}

func (q *MPSCQueue) Pop() *[]interface{} {
	q.clearOrShrinkOut()
	q.mu.Lock()
	for len(q.in) == 0 {
		q.cond.Wait()
	}
	q.in, q.out = q.out, q.in
	q.mu.Unlock()
	return &q.out
}

func (q *MPSCQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.in)
}

func (q *MPSCQueue) clearOrShrinkOut() {
	if q.shrinkSize == 0 || cap(q.out) < q.shrinkSize {
		q.out = q.out[0:0]
	} else {
		q.out = []interface{}{}
	}
}
