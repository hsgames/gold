package event

import (
	"slices"
	"sync"
)

var pool = sync.Pool{New: func() any { return &Listener{} }}

type (
	Event   uint16
	Handler func(Event, any) error
)

type Listener struct {
	e Event
	h Handler
}

type Manager struct {
	listeners map[Event][]*Listener
}

func (m *Manager) init() {
	if m.listeners == nil {
		m.listeners = make(map[Event][]*Listener)
	}
}

func (m *Manager) AddListener(e Event, handler Handler) *Listener {
	m.init()

	l := pool.Get().(*Listener)
	l.e = e
	l.h = handler

	m.listeners[e] = append(m.listeners[e], l)

	return l
}

func (m *Manager) RemoveListener(l *Listener) {
	m.init()

	if _, ok := m.listeners[l.e]; ok {
		for i, v := range m.listeners[l.e] {
			if v == l {
				slices.Delete(m.listeners[l.e], i, i+1)
				pool.Put(l)

				if len(m.listeners[l.e]) == 0 {
					delete(m.listeners, l.e)
				}

				return
			}
		}
	}
}

func (m *Manager) Dispatch(e Event, msg any) (err error) {
	m.init()

	if ls, ok := m.listeners[e]; ok {
		for _, v := range ls {
			if err = v.h(e, msg); err != nil {
				return
			}
		}
	}

	return nil
}
