package crawler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/purrice/prawler/internal/events"
	"github.com/purrice/prawler/internal/frontier"
	"github.com/purrice/prawler/internal/repository"
)

type Fetcher interface {
	Fetch(endpoint url.URL) (*http.Response, error)
	FetchWithContext(ctx context.Context, endpoint url.URL) (*http.Response, error)
}

type Crawler struct {
	agent         string
	webRecordRepo repository.WebsiteRepository
	fetcher       Fetcher
	client        frontier.Client
}

func NewCrawler(
	userAgent string,
	webRecordRepo repository.WebsiteRepository,
	fetcher Fetcher,
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
