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

	now := time.Now()

	var timeout []Node

	for _, node := range h.nodes {
		timeSinceLastseen := now.Sub(node.LastSeen)

		switch {
		case timeSinceLastseen < h.timeout:
			node.Status = Alive
		case timeSinceLastseen < h.grace:
			node.Status = Unconscious
		default:
			node.Status = Dead
			timeout = append(timeout, *node)
		}

	}

	h.mu.Unlock()

	for _, node := range timeout {
		go h.handlers.onTimeout(node)
	}
}
