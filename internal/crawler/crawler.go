package crawler

import (
	"encoding/json"
	"log"
	"net/url"

	"github.com/purrice/prawler/internal/events"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/frontier"
	"github.com/purrice/prawler/internal/repository"
)

type Crawler struct {
	agent         string
	webRecordRepo repository.WebRecordRepository
	fetcher       *fetch.Fetcher
	client        frontier.Client
}

func NewCrawler(
	userAgent string,
	webRecordRepo repository.WebRecordRepository,
	fetcher *fetch.Fetcher,
	frontier frontier.Client,
) Crawler {
	return Crawler{
		agent:         userAgent,
		webRecordRepo: webRecordRepo,
		fetcher:       fetcher,
		client:        frontier,
	}
}

func (c Crawler) handleProduceEvent(payload events.URIPayload) error {
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
	var event events.CrawlEvent

	err := json.Unmarshal(data, &event)

	if err != nil {
		return err
	}

	if err := event.IsValid(); err != nil {
		return err
	}

	switch event.Type {
	case events.CrawlURI:
		return c.handleProduceEvent(event.Payload)

	}
	return nil
}
