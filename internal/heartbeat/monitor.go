package heartbeat

import (
	"time"
)

func (h *Holter) Monitor() {
	ticker := time.NewTicker(h.MonitorPeriod)
	defer ticker.Stop()
	for range ticker.C {
		h.reconcile()
	}
}

func (h *Holter) reconcile() {
	h.mu.Lock()
	for uuid, lastseen := range h.nodes {
		if time.Since(lastseen) > h.Timeout {
			go h.handlers.onTimeout(uuid)
			delete(h.nodes, uuid)
		}
	}
	h.mu.Unlock()
}
