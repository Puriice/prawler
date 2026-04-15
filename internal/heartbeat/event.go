package heartbeat

import "log"

type EventType string

const (
	EventStatusChange EventType = "status.change"
	EventTimeout      EventType = "timeout"
)

type Event struct {
	Type EventType
	Node Node
}

func (h *Holter) OnChange(fn func(Node)) {
	h.onChangeHandler[1] = fn
}

func (h *Holter) OnTimeout(fn func(Node)) {
	h.onTimeoutHandler[1] = fn
}

func (h *Holter) triggerStateChange(node Node) {
	h.enqueue(Event{
		Type: EventStatusChange,
		Node: node,
	})
}

func (h *Holter) triggerTimeout(node Node) {
	h.enqueue(Event{
		Type: EventTimeout,
		Node: node,
	})
}

func (h *Holter) startEventWorker() {
	go func() {
		for event := range h.queue {
			h.processEvent(event)
		}
	}()
}

func (h *Holter) enqueue(e Event) {
	select {
	case h.queue <- e:
	default:
		// queue full → drop or log
		log.Println("⚠️ Event dropped:", e)
	}
}

func (h *Holter) processEvent(event Event) {
	switch event.Type {

	case EventStatusChange:
		h.mu.Lock()
		handlers := h.onChangeHandler
		h.mu.Unlock()

		for _, h := range handlers {
			h(event.Node)
		}

	case EventTimeout:
		h.mu.Lock()
		handlers := h.onTimeoutHandler
		h.mu.Unlock()

		for _, h := range handlers {
			h(event.Node)
		}
	}
}
