package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/backoff"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/enum"
	"github.com/purrice/prawler/internal/events"
	"github.com/purrice/prawler/internal/frontier"
	"github.com/purrice/prawler/internal/html"
	"github.com/purrice/prawler/internal/planner"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/uri"
	"github.com/purrice/prawler/internal/worker"
	"github.com/rabbitmq/amqp091-go"
)

type Fetcher interface {
	Fetch(endpoint url.URL) (*http.Response, error)
	FetchWithContext(ctx context.Context, endpoint url.URL) (*http.Response, error)
}

type Crawler struct {
	ctx    context.Context
	config *config.Config

	uuid  string
	agent string

	broker *messaging.RabbitBroker

	websiteRepository repository.WebsiteRepository
	fetcher           Fetcher
	client            frontier.Client

	backoff    *backoff.Manager
	backoffKey string
	planner    *planner.Planner[string, int]
	worker     *worker.WorkerManager
}

func NewCrawler(
	ctx context.Context,
	uuid string,
	userAgent string,
	webRecordRepo repository.WebsiteRepository,
	fetcher Fetcher,
	frontier frontier.Client,
	worker *worker.WorkerManager,
) *Crawler {
	planner := planner.NewPlanner[string, int]()

	for i := range worker.Count() {
		planner.AddResource(i)
	}

	config := config.GetConfig()

	return &Crawler{
		ctx:    ctx,
		config: config,

		uuid:  uuid,
		agent: userAgent,

		websiteRepository: webRecordRepo,
		fetcher:           fetcher,
		client:            frontier,

		backoff: backoff.NewManager(),
		worker:  worker,
		planner: planner,
	}
}

func (c *Crawler) Setup(rabbit *messaging.RabbitMQ) {
	broker, err := rabbit.NewBroker(c.config.ExchangeName.URI)
	c.broker = broker

	key := fmt.Sprintf("%s.%s", c.config.QueueName.URI, c.uuid)

	args := amqp091.Table{
		"x-dead-letter-exchange":    c.config.ExchangeName.URI,
		"x-dead-letter-routing-key": key,
	}

	c.backoffKey = fmt.Sprintf("%s.%s", c.config.QueueName.Backoff, c.uuid)
	listenerConfig := messaging.NewRabbitListenerConfig(
		c.backoffKey,
		c.backoffKey,
	)
	listenerConfig.Args = args

	_, err = broker.NewListenerWithConfig(
		listenerConfig,
	)

	if err != nil {
		log.Println(err)
	}
}

func (c *Crawler) publishToBackoff(delay time.Duration, payload events.CrawlPayload) error {
	e := events.CrawlEvent{
		Type:    events.CrawlURI,
		Payload: payload,
	}

	bytes, err := json.Marshal(e)

	if err != nil {
		return err
	}

	return c.broker.PublishRaw(
		c.backoffKey,
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        bytes,
			Expiration:  strconv.FormatInt(delay.Milliseconds(), 10),
		},
	)
}

func (c *Crawler) handleCrawlEvent(payload events.CrawlPayload) error {
	log.Printf("Recieve uri: %s", *payload.URI)
	targetURL, err := url.Parse(*payload.URI) // Already normalized

	if err != nil {
		return err
	}

	origin := uri.OriginKey(*targetURL).String()
	delay := max(c.backoff.NextDelay(origin), c.backoff.NextDelay(*payload.URI))

	if delay > 0 {
		return c.publishToBackoff(delay, payload)
	}

	resp, err := c.fetcher.Fetch(*targetURL)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	finalURI := resp.Request.URL

	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable:
		attempt := c.backoff.Attempt(origin)

		if attempt+1 >= c.config.CrawlingPolicy.MaximumCrawlingAttempt {
			c.websiteRepository.SetPageStatus(c.ctx, payload.PageUUID, enum.Page.Skipped)

			switch resp.StatusCode {
			case http.StatusTooManyRequests:
				log.Printf("Maximum attempt exceeded with status %d: Retry %s after 1 hour for this domain.", resp.StatusCode, targetURL)
				c.backoff.Reset(origin)
				c.backoff.Set(origin, time.Hour, attempt+1)

				c.publishToBackoff(time.Hour, payload)
				return c.client.BackoffDomain(targetURL.String(), attempt, time.Hour)
			case http.StatusServiceUnavailable:
				log.Printf("Maximum attempt exceeded with status %d: Retry %s after 1 hour for this page.", resp.StatusCode, targetURL)
				c.backoff.Reset(targetURL.String())
				c.backoff.Set(targetURL.String(), time.Hour, attempt+1)

				c.publishToBackoff(time.Hour, payload)
				return c.client.BackoffPage(payload.PageUUID, targetURL.String(), payload.Depth, attempt, time.Hour)
			}
			return nil
		}

		retryAfter := html.ParseRetryAfter(resp)
		delay := c.backoff.Add(targetURL.String(), resp.StatusCode, retryAfter)

		log.Printf("[Attempt #%d] %s backed off until %s", attempt, *payload.URI, time.Now().Add(delay).Format(time.RFC3339))
		c.websiteRepository.SetPageStatus(c.ctx, payload.PageUUID, enum.Page.Failed)
		c.publishToBackoff(delay, payload)
		return c.client.BackoffPage(payload.PageUUID, targetURL.String(), payload.Depth, attempt, delay)
	case resp.StatusCode != 200:
		return c.client.SkipCrawl(payload.PageUUID, resp.StatusCode, *payload.URI, finalURI.String(), payload.Depth)
	}

	c.backoff.Reset(origin)
	c.backoff.Reset(targetURL.String())
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
		return c.client.ConfirmCrawled(
			payload.PageUUID,
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
		resp.StatusCode,
		*payload.URI,
		finalURI.String(),
		page.CanonicalURL,
		payload.Depth,
	)
}

func (c *Crawler) Handle(data []byte) error {
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

		siteKey := uri.Origin(*url)

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

func (c *Crawler) Run() error {
	qName := fmt.Sprintf("%s.%s", c.config.QueueName.URI, c.uuid)

	log.Printf("Listening to: %s", qName)
	hostListenerConfig := messaging.NewRabbitListenerConfig(qName, qName)
	hostListener, err := c.broker.NewListenerWithConfig(hostListenerConfig)

	if err != nil {
		return err
	}

	log.Println("Start listening to hosts producing events.")
	if err := hostListener.Subscribe(c.ctx, c.Handle); err != nil {
		log.Println(err)
	}

	return nil
}
