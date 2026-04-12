package heartbeat

import (
	"sync"
	"time"
)

type Handler func(uuid string)

type handlers struct {
	onActivate Handler
	onBeats    Handler
	onTimeout  Handler
}

type Holter struct {
	MonitorPeriod time.Duration
	Timeout       time.Duration
	nodes         map[string]time.Time
	mu            sync.Mutex
	handlers      handlers
}

func noAction(uuid string) {}

func NewHolter(timeout time.Duration, monitorPeriod time.Duration) *Holter {
	return &Holter{
		MonitorPeriod: monitorPeriod,
		Timeout:       timeout,
		handlers: handlers{
			onActivate: noAction,
			onBeats:    noAction,
			onTimeout:  noAction,
		},
	}
}

func (h *Holter) OnActivate(handler Handler) {
	h.handlers.onActivate = handler
}

func (h *Holter) OnBeats(handler Handler) {
	h.handlers.onBeats = handler
}

func (h *Holter) OnTimeout(handler Handler) {
	h.handlers.onTimeout = handler
}
