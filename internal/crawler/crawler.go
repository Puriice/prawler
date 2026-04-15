package crawler

import (
	"encoding/json"
	"log"

	"github.com/purrice/prawler/internal/enum/hosts"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/types"
)

type Crawler struct {
	agent         string
	webRecordRepo repository.WebRecordRepository
	fetcher       *fetch.Fetcher
}

func NewCrawler(
	userAgent string,
	webRecordRepo repository.WebRecordRepository,
	fetcher *fetch.Fetcher,
) Crawler {
	return Crawler{
		agent:         userAgent,
		webRecordRepo: webRecordRepo,
		fetcher:       fetcher,
	}
}

func (c Crawler) handleProduceEvent(payload model.URIPayload) error {
	log.Printf("Recieve uri: %s", *payload.URI)
	// url, err := payload.GetHost()

	// if err != nil {
	// 	return err
	// }

	return nil
}

func (c Crawler) Handle(data []byte) error {
	var event model.Event

	err := json.Unmarshal(data, &event)

	if err != nil {
		return err
	}

	if err := event.IsValid(); err != nil {
		return err
	}

	payload, ok := event.Payload.(*model.URIPayload)

	if !ok {
		return types.ErrInvalidPaylod
	}

	switch *event.EventType {
	case hosts.HostProduced:
		return c.handleProduceEvent(*payload)

	}
	return nil
}
