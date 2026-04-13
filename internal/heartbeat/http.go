package heartbeat

import (
	"net/http"
	"time"

	"github.com/puriice/golibs/pkg/json"
)

type HeartbeatPayload struct {
	UUID string `json:"uuid"`
}

func (h *Holter) HandleBeats(w http.ResponseWriter, r *http.Request) {
	var payload HeartbeatPayload

	if err := json.ParseJSON(r, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.mu.Lock()

	node, exists := h.nodes[payload.UUID]

	if !exists {
		node = &Node{
			UUID:   payload.UUID,
			Status: Alive,
		}

		h.nodes[payload.UUID] = node
	}

	node.LastSeen = time.Now()
	h.mu.Unlock()

	go h.handlers.onBeats(*node)

	w.WriteHeader(http.StatusOK)
}

func (c *Holter) HandleList(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]Node, 0, len(c.nodes))

	for _, n := range c.nodes {
		result = append(result, *n)
	}

	json.SendJSON(w, 200, result)
}
