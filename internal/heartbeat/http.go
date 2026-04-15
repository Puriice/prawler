package heartbeat

import (
	"log"
	"net/http"
	"time"

	guuid "github.com/google/uuid"
	"github.com/puriice/golibs/pkg/json"
)

const (
	MAXRETRY = 2
)

type HeartbeatPayload struct {
	UUID string `json:"uuid"`
}

func (h *Holter) handleInit(w http.ResponseWriter, r *http.Request) {
	isError := true
	uuid := guuid.New()

	for range MAXRETRY {
		err := h.repo.UpdateCrawlerStatus(h.ctx, uuid.String(), string(Alive), time.Now())

		if err == nil {
			isError = false
			break
		} else {
			log.Println(err)
		}

		uuid = guuid.New()
	}

	if isError {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.registerNode(uuid.String())
	h.mu.Unlock()

	payload := HeartbeatPayload{
		UUID: uuid.String(),
	}

	json.SendJSON(w, 200, payload)

}

func (h *Holter) handleBeats(w http.ResponseWriter, r *http.Request) {
	var payload HeartbeatPayload

	if err := json.ParseJSON(r, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	node, exists := h.nodes[payload.UUID]

	if !exists {
		node = h.registerNode(payload.UUID)
	}

	node.LastSeen = time.Now()
	h.mu.Unlock()

	if !exists {
		go h.triggerStateChange(*node)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Holter) handleList(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	result := make([]Node, 0, len(h.nodes))

	for _, n := range h.nodes {
		result = append(result, *n)
	}

	json.SendJSON(w, 200, result)
}
