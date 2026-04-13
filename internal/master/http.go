package master

import (
	"context"
	"log"
	"net/http"
	"time"

	guuid "github.com/google/uuid"
	"github.com/puriice/golibs/pkg/json"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/repository"
)

const (
	MAXRETRY = 2
)

type HTTPHandler struct {
	ctx  context.Context
	repo repository.CrawlerRepository
}

type uuidPayload struct {
	Version int    `json:"uuid_version"`
	UUID    string `json:"uuid"`
}

func NewHTTPHandler(ctx context.Context, repo repository.CrawlerRepository) HTTPHandler {
	return HTTPHandler{
		ctx:  ctx,
		repo: repo,
	}
}

func (h HTTPHandler) handleUUIDRequest(w http.ResponseWriter, r *http.Request) {
	isError := true
	uuid := guuid.New()

	for range MAXRETRY {
		err := h.repo.UpdateCrawlerStatus(h.ctx, uuid.String(), string(heartbeat.Alive), time.Now())

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

	payload := uuidPayload{
		Version: 4,
		UUID:    uuid.String(),
	}

	json.SendJSON(w, 200, payload)
}

func (h HTTPHandler) RegisterHandler(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/crawler/uuid", h.handleUUIDRequest)
}
