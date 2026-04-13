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
	defer h.mu.Unlock()

	now := time.Now()

	for _, node := range h.nodes {
		timeSinceLastseen := now.Sub(node.LastSeen)

		switch {
		case timeSinceLastseen < h.timeout:
			node.Status = Alive
		case timeSinceLastseen < h.grace:
			node.Status = Unconscious
		default:
			node.Status = Dead
		}
	}
}
