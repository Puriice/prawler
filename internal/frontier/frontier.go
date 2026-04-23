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
	crawlerBroker    *messaging.RabbitBroker
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

	crawlerBroker, err := rabbit.NewBroker(config.ExchangeName.URI)

	if err != nil {
		return nil
	}
	return &FrontierNode{
		ctx:    ctx,
		config: config,

		rabbit:           rabbit,
		broker:           broker,
		crawlerBroker:    crawlerBroker,
		embbeddingBroker: embeddingBroker,

		planner:        planner.NewPlanner[string, string](),
		filter:         filter,
		parsedFilter:   filter2,
		crawlingFilter: set.NewSet[string](),

		backoff: backoff.NewManager(),
		worker:  worker.NewManager(ctx, 6, 1),
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

	key := fmt.Sprintf("%s.uuid", m.config.ExchangeName.Embedding)
	_, err := m.embbeddingBroker.NewListener(m.config.ExchangeName.Embedding, key)

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

func (m *FrontierNode) publish(crawler string, payload events.CrawlEvent) error {
	key := fmt.Sprintf("%s.%s", m.config.QueueName.URI, crawler)

	log.Printf("Publishing to: %s", key)
	return m.crawlerBroker.Publish(key, payload)
}

func (m *FrontierNode) delayPublish(crawler string, delay time.Duration, payload events.CrawlEvent) error {
	bytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s.%s", m.config.QueueName.Backoff, crawler)

	log.Printf("Publishing to: %s, Delay: %.2f", key, delay.Seconds())
	return m.crawlerBroker.PublishRaw(
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        bytes,
			Expiration:  strconv.FormatInt(delay.Milliseconds(), 10),
		},
	)
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
	siteKey := uri.Origin(*url)

	normalizedURIString := normalizedURI.String()

	domainUUID, err := m.websiteRepository.AddDomain(m.ctx, *origin)

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

	now := time.Now()

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

	delay := max(m.backoff.NextDelay(origin.String()), m.backoff.NextDelay(normalizedURIString))

	if delay > 0 {
		err = m.delayPublish(crawlerUUID, delay, event)
	} else {
		err = m.publish(crawlerUUID, event)
	}

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

func (m *FrontierNode) handleConfirmEvent(payload events.ConfirmPayload) error {
	targetURL, err := url.Parse(*payload.URI)

	if err != nil {
		return err
	}

	log.Printf("Crawled %s confirm with status %s:%d", *payload.URI, payload.Status, payload.HTTPStatus)

	m.websiteRepository.SetPageStatus(m.ctx, *payload.PageUUID, payload.Status)

	origin := uri.Origin(*targetURL)
	normalized := uri.Normalize(*targetURL)

	m.crawlingFilter.Remove(normalized.String())

	switch payload.Status {
	case enum.Page.Parsed:
		m.backoff.Reset(origin)
		m.backoff.Reset(normalized.String())

		if url, err := url.Parse(payload.FinalURI); err != nil && payload.FinalURI != "" {
			m.backoff.Reset(uri.Normalize(*url).String())
		}

		if url, err := url.Parse(payload.Canonical); err != nil && payload.Canonical != "" {
			m.backoff.Reset(uri.Normalize(*url).String())
		}

		m.embbeddingBroker.Publish(fmt.Sprintf("%s.uuid", m.config.ExchangeName.Embedding), events.EmbeddingEvent{
			Type:     events.EventEmbedding,
			PageUUID: *payload.PageUUID,
		})
		fallthrough
	case enum.Page.Skipped, enum.Page.Failed:
		m.addFilter(*payload.URI)
		m.addFilter(payload.Canonical)
		m.addFilter(payload.FinalURI)
	}

	return nil
}

func (m *FrontierNode) handleBackoffEvent(payload events.BackoffPayload) error {
	url, err := url.Parse(*payload.URI)

	if err != nil {
		return err
	}

	normalized := uri.Normalize(*url)
	m.crawlingFilter.Remove(normalized.String())
	delay := time.Until(payload.BackoffUntil)

	if delay <= 0 {
		switch payload.Type {
		case enum.Backoff.Domain:
			origin := uri.OriginKey(*url).String()
			m.backoff.Reset(origin)
		case enum.Backoff.Page:
			m.backoff.Reset(*payload.URI)
		}
		return nil
	}

	switch payload.Type {
	case enum.Backoff.Domain:
		origin := uri.OriginKey(*url).String()
		m.backoff.Set(origin, delay, payload.Attempt+1)

	case enum.Backoff.Page:
		m.websiteRepository.SetPageStatus(m.ctx, *payload.PageUUID, enum.Page.Failed)
		m.backoff.Set(*payload.URI, delay, payload.Attempt+1)

	}

	return nil
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

		if err := payload.IsValid(); err != nil {
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
