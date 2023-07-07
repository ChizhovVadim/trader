package quik

import (
	"reflect"
	"sync"
)

type Publisher interface {
	Publish(interface{})
}

type Subscriber interface {
	Subscribe(handler Handler) UnsubscribeFunc
}

type Handler interface {
	Handle(interface{})
}

type UnsubscribeFunc func()

type HandlerFunc func(interface{})

func (h HandlerFunc) Handle(msg interface{}) {
	h(msg)
}

type EventManager struct {
	mu       sync.RWMutex
	handlers []Handler
}

func (e *EventManager) Publish(msg interface{}) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, h := range e.handlers {
		h.Handle(msg)
	}
}

func (e *EventManager) Subscribe(handler Handler) UnsubscribeFunc {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, handler)
	return func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		v := reflect.ValueOf(handler)
		for i, h := range e.handlers {
			if reflect.ValueOf(h) == v {
				e.handlers = append(e.handlers[:i], e.handlers[i+1:]...)
				return
			}
		}
	}
}
