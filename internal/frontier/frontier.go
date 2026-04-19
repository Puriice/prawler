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

	rabbit *messaging.RabbitMQ

	websiteRepository repository.WebsiteRepository
	crawlerRepository repository.CrawlerRepository

	blacklists  *blacklists.Blacklists
	robotParser *robots.RobotParser
	holter      *heartbeat.Holter
	planner     *planner.Planner[string, string]
	filter      Filter

	backoff *backoff.Manager
	worker  *worker.WorkerManager
}

func NewFrontierNode(
	ctx context.Context,
	rabbit *messaging.RabbitMQ,
	filter Filter,
) *FrontierNode {
	return &FrontierNode{
		ctx:    ctx,
		config: config.GetConfig(),

		rabbit:  rabbit,
		planner: planner.NewPlanner[string, string](),
		filter:  filter,

		backoff: backoff.NewManager(backoff.Default()),
		worker:  worker.NewManager(ctx, 3),
	}
}

func (m *FrontierNode) Setup(
	crawlerRepository repository.CrawlerRepository,
	websiteRepository repository.WebsiteRepository,
	mux *http.ServeMux,
) {
	robotParser := robots.NewRobotParser(websiteRepository, fetch.NewFecter(nil))
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

	for _, page := range pages {
		m.addFilter(page.URL)
		m.addFilter(page.CanonicalURL)
	}

	err := m.rabbit.Channel.ExchangeDeclare(
		m.config.ExchangeName.Backoff,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)

	key := fmt.Sprintf("%s.uri", m.config.ExchangeName.Frontier)

	args := amqp.Table{
		"x-dead-letter-exchange":    m.config.ExchangeName.Frontier,
		"x-dead-letter-routing-key": key,
	}

	_, err = m.rabbit.Channel.QueueDeclare(
		m.config.ExchangeName.Backoff,
		true,
		false,
		false,
		false,
		args,
	)

	if err != nil {
		log.Println(err)
	}

	err = m.rabbit.Channel.QueueBind(
		m.config.ExchangeName.Backoff,
		m.config.ExchangeName.Backoff,
		m.config.ExchangeName.Backoff,
		false,
		nil,
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

func (m FrontierNode) handleURIRegister(payload events.URIPayload) error {
	url, err := url.Parse(*payload.URI)
	log.Printf("RECEIVED: %s\n", *payload.URI)

	if err != nil {
		return nil // Error parsing url return nil because we don't want a retry
	}

	if payload.Depth+1 > m.config.CrawlingPolicy.MaximumCrawlingDepth {
		return nil
	}

	origin := uri.OriginKey(*url)
	normalizedURI := uri.Normalize(*url)
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

	if payload.Source != nil && payload.Source.IsValid() == nil {
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

	return nil
}

func (m FrontierNode) addFilter(u string) {
	url, err := url.Parse(u)

	if err != nil {
		return
	}

	normalized := uri.Normalize(*url)
	m.filter.Add(normalized.String())
}

func (m FrontierNode) backingOff(p events.BackoffPayload) error {
	url, err := url.Parse(*p.URI)

	if err != nil {
		return err
	}

	sitekey := uri.SiteKey(*url)
	attempt := m.backoff.Attempt(sitekey)

	if attempt+1 >= m.config.CrawlingPolicy.MaximumCrawlingAttempt {
		log.Printf("Maximum attempt exceeded for: %s", sitekey)
		m.websiteRepository.SetPageStatus(m.ctx, *p.PageUUID, enum.Page.Skipped)
		m.blacklists.Add(*p.URI)
	} else {
		delay := m.backoff.Add(sitekey, p.HTTPStatus, p.RetryAfter)
		log.Printf("[Attempt #%d] Retry %s After %.0fs", attempt, sitekey, delay.Seconds())
		now := time.Now()

		payloadBody := events.URIPayload{
			URI:       p.URI,
			Depth:     p.Depth,
			Timestamp: &now,
		}

		bytes, err := json.Marshal(payloadBody)

		if err != nil {
			return err
		}

		payload := events.FrontierEvent{
			Type:    events.FrontierURIRegister,
			Payload: bytes,
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		return m.rabbit.Channel.Publish(
			m.config.ExchangeName.Backoff,
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

	return nil
}

func (m FrontierNode) handleConfirmEvent(payload events.ConfirmPayload) error {
	log.Printf("Crawled %s confirm with status %s:%d", *payload.URI, payload.Status, payload.HTTPStatus)
	url, err := url.Parse(*payload.URI)

	if err != nil {
		return err
	}

	m.websiteRepository.SetPageStatus(m.ctx, *payload.PageUUID, payload.Status)

	sitekey := uri.SiteKey(*url)

	switch payload.Status {
	case enum.Page.Parsed:
		m.backoff.Reset(sitekey)
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

func (m FrontierNode) handleBackoffEvent(payload events.BackoffPayload) error {
	log.Printf("Crawled %s Failed with status %d backoff for the moment", *payload.URI, payload.HTTPStatus)
	m.websiteRepository.SetPageStatus(m.ctx, *payload.PageUUID, enum.Page.Failed)

	return m.backingOff(payload)
}

func (m FrontierNode) Handle(data []byte) error {
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

func (m FrontierNode) Run() {
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

	listenerConfig := messaging.NewRabbitListenerConfig(m.config.ExchangeName.Frontier, fmt.Sprintf("%s.#", m.config.ExchangeName.Frontier))
	listener, err := broker.NewListenerWithConfig(listenerConfig)

	log.Println("Start listening to slave producing events.")
	if err := listener.Subscribe(m.ctx, m.Handle); err != nil {
		log.Println(err)
	}
}
