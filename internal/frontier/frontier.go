package frontier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/backoff"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/config/blacklists"
	"github.com/purrice/prawler/internal/enum"
	"github.com/purrice/prawler/internal/events"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/planner"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
	"github.com/purrice/prawler/internal/set"
	"github.com/purrice/prawler/internal/uri"
	"github.com/purrice/prawler/internal/worker"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrNoAvaliableCrawler = errors.New("No crawler avaliable")
)

type Filter interface {
	Add(uri string)
	Contains(uri string) bool
}
type FrontierNode struct {
	ctx    context.Context
	config *config.Config

	client Client

	rabbit           *messaging.RabbitMQ
	broker           *messaging.RabbitBroker
	backoffBroker    *messaging.RabbitBroker
	embbeddingBroker *messaging.RabbitBroker

	websiteRepository repository.WebsiteRepository
	crawlerRepository repository.CrawlerRepository

	blacklists     *blacklists.Blacklists
	robotParser    *robots.RobotParser
	holter         *heartbeat.Holter
	planner        *planner.Planner[string, string]
	filter         Filter
	parsedFilter   Filter
	crawlingFilter *set.Set[string]

	backoff *backoff.Manager
	worker  *worker.WorkerManager
}

func NewFrontierNode(
	ctx context.Context,
	rabbit *messaging.RabbitMQ,
	filter Filter,
	filter2 Filter,
) *FrontierNode {
	config := config.GetConfig()
	broker, err := rabbit.NewBroker(config.ExchangeName.Frontier)

	if err != nil {
		return nil
	}

	embeddingBroker, err := rabbit.NewBroker(config.ExchangeName.Embedding)

	if err != nil {
		return nil
	}

	backoffBroker, err := rabbit.NewBroker(config.ExchangeName.Backoff)

	if err != nil {
		return nil
	}

	return &FrontierNode{
		ctx:    ctx,
		config: config,

		rabbit:           rabbit,
		broker:           broker,
		backoffBroker:    backoffBroker,
		embbeddingBroker: embeddingBroker,

		planner:        planner.NewPlanner[string, string](),
		filter:         filter,
		parsedFilter:   filter2,
		crawlingFilter: set.NewSet[string](),

		backoff: backoff.NewManager(),
		worker:  worker.NewManager(ctx, 6, 2),
	}
}

func (m *FrontierNode) Setup(
	crawlerRepository repository.CrawlerRepository,
	websiteRepository repository.WebsiteRepository,
	mux *http.ServeMux,
) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	robotParser := robots.NewRobotParser(websiteRepository, fetch.NewFecter(client))
	blacklists := blacklists.NewBlacklist(websiteRepository)

	holter := heartbeat.NewHolter(m.ctx, 5*time.Second, 10*time.Second, 2*time.Second, crawlerRepository)
	holter.OnChange(m.handleNodeStatusChanges)
	holter.OnTimeout(m.handleNodeStatusChanges)
	holter.Run(mux)

	m.holter = holter

	m.crawlerRepository = crawlerRepository
	m.websiteRepository = websiteRepository

	m.blacklists = blacklists
	m.robotParser = &robotParser

	m.worker.SpawnWorker()

	pages := websiteRepository.GetFinishedPage(m.ctx)

	log.Printf("Load %d parsed or skipped page", len(pages))
	for _, page := range pages {
		m.addParsedFilter(page.URL)
		m.addParsedFilter(page.CanonicalURL)
	}

	brokerConfig := messaging.NewBrokerConfig()
	brokerConfig.Kind = messaging.Direct
	broker, err := m.rabbit.NewBrokerWithConfig(m.config.ExchangeName.Backoff, brokerConfig)

	if err != nil {
		return
	}

	key := fmt.Sprintf("%s.uri", m.config.ExchangeName.Frontier)

	args := amqp.Table{
		"x-dead-letter-exchange":    m.config.ExchangeName.Frontier,
		"x-dead-letter-routing-key": key,
	}

	listenerConfig := messaging.NewRabbitListenerConfig(
		m.config.ExchangeName.Backoff,
		m.config.ExchangeName.Backoff,
	)
	listenerConfig.Args = args

	_, err = broker.NewListenerWithConfig(
		listenerConfig,
	)

	if err != nil {
		log.Println(err)
	}
}

func (m *FrontierNode) handleNodeStatusChanges(node heartbeat.Node) {
	switch node.Status {
	case heartbeat.Alive:
		log.Printf("Add %s to the planner.", node.UUID)
		m.planner.AddResource(node.UUID)
	case heartbeat.Unconscious, heartbeat.Dead:
		log.Printf("Remove %s from the planner.", node.UUID)
		m.planner.RemoveResource(node.UUID)
	}
}

func (m *FrontierNode) handleURIRegister(payload events.URIPayload) error {
	url, err := url.Parse(*payload.URI)
	log.Printf("RECEIVED: %s\n", *payload.URI)

	if err != nil {
		return nil // Error parsing url return nil because we don't want a retry
	}

	normalizedURI := uri.Normalize(*url)

	if payload.Depth+1 > m.config.CrawlingPolicy.MaximumCrawlingDepth {
		log.Printf("Maximum depth reached: %s", normalizedURI.String())
		return nil
	}

	origin := uri.OriginKey(*url)
	siteKey := uri.SiteKey(*url)

	normalizedURIString := normalizedURI.String()

	domainUUID, err := m.websiteRepository.AddDomain(m.ctx, origin)

	if err != nil {
		return err
	}

	pageUUID, err := m.websiteRepository.AddPage(m.ctx, domainUUID, *normalizedURI, payload.Depth)

	if err != nil {
		return err
	}

	if payload.Source != nil && payload.Source.IsValid() == nil && payload.Source.AnchorText != "" {
		err := m.websiteRepository.AddLink(m.ctx, *payload.Source.PageUUID, pageUUID, payload.Source.AnchorText)

		if err != nil {
			log.Println(err)
		}
	}

	if m.blacklists.Contains(siteKey) {
		return nil
	}

	if m.filter.Contains(normalizedURIString) {
		return nil
	}

	if m.parsedFilter.Contains(normalizedURIString) {
		m.filter.Add(normalizedURIString)
		links := m.websiteRepository.GetLinks(m.ctx, pageUUID)
		now := time.Now()

		depth := payload.Depth + 1
		source := &events.Source{
			PageUUID:   &pageUUID,
			AnchorText: "",
		}
		key := fmt.Sprintf("%s.uri", m.config.ExchangeName.Frontier)

		for _, link := range links {
			p := events.URIPayload{
				URI:       &link,
				Revisit:   true,
				Depth:     depth,
				Source:    source,
				Timestamp: &now,
			}

			bytes, err := json.Marshal(p)

			if err != nil {
				log.Println(err)
				continue
			}

			event := events.FrontierEvent{
				Type:    events.FrontierURIRegister,
				Payload: bytes,
			}

			m.broker.Publish(key, event)
		}

		return nil
	}

	if m.crawlingFilter.Contains(normalizedURIString) {
		return nil
	}

	rbs, err := m.robotParser.Parse(*url)

	if err != nil {
		return nil
	}

	if !rbs.IsAllow(m.config.CrawlingPolicy.UserAgent, normalizedURIString) && url.Hostname() != "localhost" {
		return nil
	}

	crawlerUUID, ok := m.planner.Plan(siteKey)

	if !ok {
		return ErrNoAvaliableCrawler
	}

	broker, err := m.rabbit.NewBroker(m.config.ExchangeName.URI)

	if err != nil {
		return err
	}

	now := time.Now()
	key := fmt.Sprintf("%s.%s", m.config.QueueName, crawlerUUID)

	event := events.CrawlEvent{
		Type: events.CrawlURI,
		Payload: events.CrawlPayload{
			URIPayload: events.URIPayload{
				URI:       &normalizedURIString,
				Depth:     payload.Depth,
				Timestamp: &now,
			},
			PageUUID: pageUUID,
		},
	}

	log.Printf("Publishing to: %s", key)
	err = broker.Publish(key, event)

	if err != nil {
		return err
	}

	m.crawlingFilter.Add(normalizedURIString)

	return nil
}

func (m *FrontierNode) addFilter(u string) {
	url, err := url.Parse(u)

	if err != nil {
		return
	}

	normalized := uri.Normalize(*url)
	m.filter.Add(normalized.String())
}

func (m *FrontierNode) addParsedFilter(u string) {
	url, err := url.Parse(u)

	if err != nil {
		return
	}

	normalized := uri.Normalize(*url)
	m.parsedFilter.Add(normalized.String())
}

func (m *FrontierNode) delayPublish(delay time.Duration, payload events.URIPayload) error {
	bytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	e := events.FrontierEvent{
		Type:    events.FrontierURIRegister,
		Payload: bytes,
	}

	body, err := json.Marshal(e)
	if err != nil {
		return err
	}

	return m.backoffBroker.PublishRaw(
		m.config.ExchangeName.Backoff,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Expiration:  strconv.FormatInt(delay.Milliseconds(), 10),
		},
	)
}

func (m *FrontierNode) backingOff(p events.BackoffPayload) error {
	url, err := url.Parse(*p.URI)

	if err != nil {
		return err
	}

	sitekey := uri.SiteKey(*url)
	attempt := m.backoff.Attempt(sitekey)

	now := time.Now()
	payload := events.URIPayload{
		URI:       p.URI,
		Depth:     p.Depth,
		Timestamp: &now,
	}

	if attempt+1 >= m.config.CrawlingPolicy.MaximumCrawlingAttempt {
		m.websiteRepository.SetPageStatus(m.ctx, *p.PageUUID, enum.Page.Skipped)

		switch p.HTTPStatus {
		case 403:
			log.Printf("Maximum attempt exceeded with status %d: blacklist %s", p.HTTPStatus, *p.URI)
			m.blacklists.Add(*p.URI)
		case 400, 401, 402, 404:
			log.Printf("Maximum attempt exceeded with status %d: Skipping this page %s", p.HTTPStatus, *p.URI)
			return m.websiteRepository.SetPageStatus(m.ctx, *p.PageUUID, enum.Page.Skipped)
		case 429:
			log.Printf("Maximum attempt exceeded with status %d: Retry %s after 1 hour for this domain.", p.HTTPStatus, *p.URI)
			m.backoff.Reset(sitekey)
			m.backoff.Set(sitekey, time.Hour)
		case 503:
			log.Printf("Maximum attempt exceeded with status %d: Retry %s after 1 hour for this page.", p.HTTPStatus, *p.URI)
			m.websiteRepository.SetPageStatus(m.ctx, *p.PageUUID, enum.Page.Failed)
			return m.delayPublish(time.Hour, payload)
		}
	} else {
		delay := m.backoff.Add(sitekey, p.HTTPStatus, p.RetryAfter)
		log.Printf("[Attempt #%d] Retry %s After %.0fs", attempt, sitekey, delay.Seconds())

		return m.delayPublish(delay, payload)
	}

	return nil
}

func (m *FrontierNode) handleConfirmEvent(payload events.ConfirmPayload) error {
	url, err := url.Parse(*payload.URI)

	if err != nil {
		return err
	}

	log.Printf("Crawled %s confirm with status %s:%d", *payload.URI, payload.Status, payload.HTTPStatus)

	m.websiteRepository.SetPageStatus(m.ctx, *payload.PageUUID, payload.Status)

	sitekey := uri.SiteKey(*url)
	normalized := uri.Normalize(*url)

	m.crawlingFilter.Remove(normalized.String())

	switch payload.Status {
	case enum.Page.Parsed:
		m.backoff.Reset(sitekey)
		m.embbeddingBroker.Publish(fmt.Sprintf("%s.uuid", m.config.ExchangeName.Embedding), events.EmbeddingEvent{
			Type:     events.EventEmbedding,
			PageUUID: *payload.PageUUID,
		})
		fallthrough
	case enum.Page.Skipped:
		m.addFilter(*payload.URI)
		m.addFilter(payload.Canonical)
		m.addFilter(payload.FinalURI)
	case enum.Page.Failed:
		return m.backingOff(
			events.BackoffPayload{
				PageUUID: payload.PageUUID,
				Depth:    payload.Depth,

				URI:        &payload.FinalURI,
				HTTPStatus: payload.HTTPStatus,
				RetryAfter: 0,

				Timestamp: payload.Timestamp,
			},
		)
	}

	return nil
}

func (m *FrontierNode) handleBackoffEvent(payload events.BackoffPayload) error {
	url, err := url.Parse(*payload.URI)

	if err != nil {
		return err
	}

	log.Printf("Crawled %s Failed with status %d backoff for the moment", *payload.URI, payload.HTTPStatus)
	m.websiteRepository.SetPageStatus(m.ctx, *payload.PageUUID, enum.Page.Failed)

	normalized := uri.Normalize(*url)
	m.crawlingFilter.Remove(normalized.String())

	return m.backingOff(payload)
}

func (m *FrontierNode) Handle(data []byte) error {
	var event events.FrontierEvent

	err := json.Unmarshal(data, &event)

	if err != nil {
		log.Println(err)
		return err
	}

	if err := event.IsValid(); err != nil {
		log.Println(err)
		return err
	}
	log.Printf("Event Incomming: %s", event.Type)

	switch event.Type {
	case events.FrontierURIRegister:
		var payload events.URIPayload

		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			log.Println(err)
			return nil
		}

		err = m.worker.AssignTo(0, func() {
			if err := m.handleURIRegister(payload); err != nil {
				log.Println(err)
			}
		})

		if err != nil {
			log.Println(err)
		}
	case events.FrontierCrawlConfirm:
		var payload events.ConfirmPayload

		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			log.Println(err)
			return nil
		}

		err = m.worker.AssignTo(1, func() {
			if err := m.handleConfirmEvent(payload); err != nil {
				log.Println(err)
			}
		})

		if err != nil {
			log.Println(err)
		}
	case events.FrontierBackoff:
		var payload events.BackoffPayload

		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			log.Println(err)
			return nil
		}

		err = m.worker.AssignTo(2, func() {
			if err := m.handleBackoffEvent(payload); err != nil {
				log.Println(err)
			}
		})

		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (m *FrontierNode) Run() {
	broker, err := m.rabbit.NewBroker(m.config.ExchangeName.Frontier)

	if err != nil {
		log.Fatal(err)
	}

	crawlers, err := m.crawlerRepository.QueryCrawlerStatus(m.ctx)

	if err != nil {
		log.Fatal(err)
	}

	m.holter.Load(func() map[string]*heartbeat.Node {
		nodes := make(map[string]*heartbeat.Node, len(crawlers))

		for _, crawler := range crawlers {
			nodes[crawler.UUID] = &heartbeat.Node{
				UUID:     crawler.UUID,
				Status:   heartbeat.Status(crawler.Status),
				LastSeen: crawler.LastSeen,
			}
		}

		return nodes
	})

	keys := keys{
		Register: fmt.Sprintf("%s.uri", m.config.ExchangeName.Frontier),
		Confirm:  fmt.Sprintf("%s.confirm", m.config.ExchangeName.Frontier),
		Backoff:  fmt.Sprintf("%s.backoff", m.config.ExchangeName.Frontier),
	}

	uriListenerConfig := messaging.NewRabbitListenerConfig(keys.Register, keys.Register)
	confirmListenerConfig := messaging.NewRabbitListenerConfig(keys.Confirm, keys.Confirm)
	backoffListenerConfig := messaging.NewRabbitListenerConfig(keys.Backoff, keys.Backoff)

	uriListenerConfig.PrefetchCount = 1
	confirmListenerConfig.PrefetchCount = 1
	backoffListenerConfig.PrefetchCount = 1

	m.worker.AssignTo(3, func() {
		listener, err := broker.NewListenerWithConfig(uriListenerConfig)

		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("Start listening to slave producing %s events.", keys.Register)
		if err := listener.Subscribe(m.ctx, m.Handle); err != nil {
			log.Println(err)
		}
	})

	m.worker.AssignTo(4, func() {
		listener, err := broker.NewListenerWithConfig(confirmListenerConfig)

		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("Start listening to slave producing %s events.", keys.Confirm)
		if err := listener.Subscribe(m.ctx, m.Handle); err != nil {
			log.Println(err)
		}
	})

	m.worker.AssignTo(5, func() {
		listener, err := broker.NewListenerWithConfig(backoffListenerConfig)

		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("Start listening to slave producing %s events.", keys.Backoff)
		if err := listener.Subscribe(m.ctx, m.Handle); err != nil {
			log.Println(err)
		}
	})
}
