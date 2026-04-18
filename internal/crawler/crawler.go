package crawler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/purrice/prawler/internal/enum"
	"github.com/purrice/prawler/internal/events"
	"github.com/purrice/prawler/internal/frontier"
	"github.com/purrice/prawler/internal/html"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/worker"
)

type Fetcher interface {
	Fetch(endpoint url.URL) (*http.Response, error)
	FetchWithContext(ctx context.Context, endpoint url.URL) (*http.Response, error)
}

type Crawler struct {
	ctx               context.Context
	agent             string
	websiteRepository repository.WebsiteRepository
	fetcher           Fetcher
	client            frontier.Client
	worker            *worker.WorkerManager
}

func NewCrawler(
	ctx context.Context,
	userAgent string,
	webRecordRepo repository.WebsiteRepository,
	fetcher Fetcher,
	frontier frontier.Client,
	worker *worker.WorkerManager,
) Crawler {
	return Crawler{
		ctx:               ctx,
		agent:             userAgent,
		websiteRepository: webRecordRepo,
		fetcher:           fetcher,
		client:            frontier,
		worker:            worker,
	}
}

func (c Crawler) handleCrawlEvent(payload events.URIPayload) error {
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

	finalURI := resp.Request.URL

	parser, err := html.NewParser(finalURI.String())

	if err != nil {
		return err
	}

	meta, page, content, err := parser.ParseReader(resp.Body)

	if err != nil {
		return err
	}

	pageUUID, err := c.websiteRepository.AddPage(c.ctx, *finalURI, payload.Depth, *page)

	if err == nil {
		c.websiteRepository.AddPageMetadata(c.ctx, pageUUID, *meta)
		c.websiteRepository.AddPageContent(c.ctx, pageUUID, *content)
	}

	if page.NoFollow {
		c.client.ConfirmCrawled(pageUUID, enum.Page.Parsed, *payload.URI, finalURI.String(), page.CanonicalURL, payload.Depth)
		return nil
	}

	for _, link := range page.Links {
		if link.IsNoFollow {
			continue
		}

		c.client.Register(link.TargetURL)
	}

	c.client.ConfirmCrawled(pageUUID, enum.Page.Parsed, *payload.URI, finalURI.String(), page.CanonicalURL, payload.Depth)
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
		c.worker.Assign(func() {
			err := c.handleCrawlEvent(event.Payload)

			if err != nil {
				log.Println(err)
			}
		})

	}
	return nil
}
