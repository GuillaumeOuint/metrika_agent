package watch

import "sync"

type Watcher interface {
	StartUnsafe()
	Stop()

	Subscribe(chan<- interface{})

	once() *sync.Once
}

func Start(watcher Watcher) {
	watcher.once().Do(watcher.StartUnsafe)
}

type Watch struct {
	Running bool

	StartFn   func()
	StopFn    func()
	StopKey   chan bool
	startOnce *sync.Once

	listeners []chan<- interface{}
}

func NewWatch() Watch {
	return Watch{
		Running:   false,
		StartFn:   func() {},
		StopFn:    func() {},
		StopKey:   make(chan bool, 1),
		startOnce: &sync.Once{},
	}
}

func (w *Watch) StartUnsafe() {
	w.Running = true
}

func (w *Watch) Stop() {
	if !w.Running {
		return
	}
	w.Running = false

	w.StopFn()
	w.StopKey <- true
}

func (w *Watch) once() *sync.Once {
	return w.startOnce
}

// Subscription mechanism

func (w *Watch) Subscribe(handler chan<- interface{}) {
	w.listeners = append(w.listeners, handler)
}

func (w *Watch) Emit(message interface{}) {
	for _, handler := range w.listeners {
		handler <- message
	}
}
