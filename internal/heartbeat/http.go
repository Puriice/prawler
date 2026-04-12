package heartbeat

import (
	"net/http"
	"time"

	"github.com/puriice/golibs/pkg/json"
)

type HeartbeatPayload struct {
	UUID string `json:"uuid"`
}

func (h *Holter) HandleFunc(w http.ResponseWriter, r *http.Request) {
	var payload HeartbeatPayload

	if err := json.ParseJSON(r, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.nodes[payload.UUID] = time.Now()

	go h.handlers.onBeats(payload.UUID)

	w.WriteHeader(http.StatusOK)
}
