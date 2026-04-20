package crawler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/purrice/prawler/internal/enum"
	"github.com/purrice/prawler/internal/events"
	"github.com/purrice/prawler/internal/frontier"
	"github.com/purrice/prawler/internal/html"
	"github.com/purrice/prawler/internal/planner"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/uri"
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

	planner *planner.Planner[string, int]
	worker  *worker.WorkerManager
}

func NewCrawler(
	ctx context.Context,
	userAgent string,
	webRecordRepo repository.WebsiteRepository,
	fetcher Fetcher,
	frontier frontier.Client,
	worker *worker.WorkerManager,
) Crawler {
	planner := planner.NewPlanner[string, int]()

	for i := range worker.Count() {
		planner.AddResource(i)
	}

	return Crawler{
		ctx:               ctx,
		agent:             userAgent,
		websiteRepository: webRecordRepo,
		fetcher:           fetcher,
		client:            frontier,

		worker:  worker,
		planner: planner,
	}
}

func (c Crawler) handleCrawlEvent(payload events.CrawlPayload) error {
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

	switch {
	case resp.StatusCode == 429 || resp.StatusCode >= 500:
		return c.client.Backoff(payload.PageUUID, *payload.URI, resp.StatusCode, html.ParseRetryAfter(resp))
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return c.client.FailedCrawled(payload.PageUUID, resp.StatusCode, *payload.URI, finalURI.String(), payload.Depth)
	case resp.StatusCode != 200:
		return c.client.SkipCrawl(payload.PageUUID, resp.StatusCode, *payload.URI, finalURI.String(), payload.Depth)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	if !strings.Contains(contentType, "text/html") {
		return c.client.SkipCrawl(payload.PageUUID, resp.StatusCode, *payload.URI, finalURI.String(), payload.Depth)
	}

	parser, err := html.NewParser(finalURI.String())

	if err != nil {
		return c.client.FailedCrawled(payload.PageUUID, resp.StatusCode, *payload.URI, finalURI.String(), payload.Depth)
	}

	meta, page, content, err := parser.ParseReader(resp.Body)

	if err != nil {
		return c.client.FailedCrawled(payload.PageUUID, resp.StatusCode, *payload.URI, finalURI.String(), payload.Depth)
	}

	err = c.websiteRepository.AddPageInformation(c.ctx, payload.PageUUID, *finalURI, payload.Depth, *page)

	if err == nil {
		c.websiteRepository.AddPageMetadata(c.ctx, payload.PageUUID, *meta)
		c.websiteRepository.AddPageContent(c.ctx, payload.PageUUID, *content)
	}

	if page.NoFollow {
		return c.client.ConfirmCrawled(payload.PageUUID,
			enum.Page.Parsed,
			resp.StatusCode,
			*payload.URI,
			finalURI.String(),
			page.CanonicalURL,
			payload.Depth,
		)
	}

	for _, link := range page.Links {
		if link.IsNoFollow {
			continue
		}

		c.client.Register(link.TargetURL, payload.Depth+1, &events.Source{
			PageUUID:   &payload.PageUUID,
			AnchorText: link.AnchorText,
		})
	}

	return c.client.ConfirmCrawled(
		payload.PageUUID,
		enum.Page.Parsed,
		resp.StatusCode,
		*payload.URI,
		finalURI.String(),
		page.CanonicalURL,
		payload.Depth,
	)
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

	if err := event.Payload.IsValid(); err != nil {
		return err
	}

	switch event.Type {
	case events.CrawlURI:
		url, err := url.Parse(*event.Payload.URI)

		if err != nil {
			return nil
		}

		siteKey := uri.SiteKey(*url)

		workerId, ok := c.planner.Plan(siteKey)

		if !ok {
			log.Println(worker.ErrMaximumWorkerCapacity)
			return worker.ErrMaximumWorkerCapacity
		}

		err = c.worker.AssignTo(workerId, func() {
			if err := c.handleCrawlEvent(event.Payload); err != nil {
				log.Println(err)
			}
		})

		if err != nil {
			log.Println(err)
		}
	}

	return nil
}
