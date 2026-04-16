package crawler

import (
	"encoding/json"
	"log"
	"net/url"

	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/repository"
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
	uri, err := url.Parse(*payload.URI)

	if err != nil {
		return err
	}

	resp, err := c.fetcher.Fetch(*uri)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// finalURI := resp.Request.URL

	return nil
}

func (c Crawler) Handle(data []byte) error {
	var event Event

	err := json.Unmarshal(data, &event)

	if err != nil {
		return err
	}

	if err := event.IsValid(); err != nil {
		return err
	}

	switch event.Type {
	case EventURI:
		return c.handleProduceEvent(event.Payload)

	}
	return nil
}
