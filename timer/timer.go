package timer

import (
	"container/heap"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/safe"
	"time"
)

type Timer struct {
	f        func()
	id       uint64
	end      time.Time
	interval time.Duration
	index    int
}

type Queue []*Timer

func (q Queue) Len() int {
	return len(q)
}

func (q Queue) Less(i, j int) bool {
	return q[i].end.Before(q[j].end)
}

func (q Queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *Queue) Push(x interface{}) {
	n := len(*q)
	timer := x.(*Timer)
	timer.index = n
	*q = append(*q, timer)
}

func (q *Queue) Pop() interface{} {
	old := *q
	n := len(old)
	timer := old[n-1]
	old[n-1] = nil
	timer.index = -1
	*q = old[0 : n-1]
	return timer
}

type Manager struct {
	id     uint64
	q      Queue
	fs     []func()
	logger log.Logger
}

func NewManager(logger log.Logger) *Manager {
	return &Manager{logger: logger}
}

func (m *Manager) AddTimer(d time.Duration, f func()) uint64 {
	m.id++
	timer := &Timer{
		f:   f,
		id:  m.id,
		end: time.Now().Add(d),
	}
	heap.Push(&m.q, timer)
	return timer.id
}

func (m *Manager) AddTicker(d time.Duration, f func()) uint64 {
	m.id++
	timer := &Timer{
		f:        f,
		id:       m.id,
		end:      time.Now().Add(d),
		interval: d,
	}
	heap.Push(&m.q, timer)
	return timer.id
}

func (m *Manager) Remove(id uint64) {
	for _, timer := range m.q {
		if timer.id == id {
			heap.Remove(&m.q, timer.index)
			return
		}
	}
}

func (m *Manager) RemoveAll() {
	m.id = 0
	m.q = Queue{}
	m.fs = []func(){}
}

func (m *Manager) Run(limit int) {
	var count int
	now := time.Now()
	for len(m.q) > 0 {
		timer := m.q[0]
		if !timer.end.After(now) {
			m.fs = append(m.fs, timer.f)
			if timer.interval > 0 {
				timer.end = timer.end.Add(timer.interval)
				heap.Fix(&m.q, timer.index)
			} else {
				heap.Pop(&m.q)
			}
		} else {
			break
		}
		count++
		if limit > 0 && count >= limit {
			break
		}
	}
	if len(m.fs) > 0 {
		for i, f := range m.fs {
			m.fs[i] = nil
			func() {
				defer safe.Recover(m.logger)
				f()
			}()
		}
		m.fs = m.fs[:0]
	}
}
