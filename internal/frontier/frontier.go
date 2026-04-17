package frontier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/config/blacklists"
	"github.com/purrice/prawler/internal/events"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/frontier/planner"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
	"github.com/purrice/prawler/internal/uri"
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
}

func NewFrontierNode(
	ctx context.Context,
	db *pgxpool.Pool,
	rabbit *messaging.RabbitMQ,
	filter Filter,
) FrontierNode {
	fetcher := fetch.NewFecter(nil)

	crawlerRepository := repository.NewPostgresCrawlerRepository(db)
	websiteRepository := repository.NewPostgresWebsiteRepository(db)
	blacklistRepository := repository.NewPostgresBlacklistRepository(db)

	robotParser := robots.NewRobotParser(websiteRepository, fetcher)
	blacklists := blacklists.NewBlacklist(blacklistRepository)
	holter := heartbeat.NewHolter(ctx, 5*time.Second, 10*time.Second, 2*time.Second, crawlerRepository)

	config := config.GetConfig()

	frontier := FrontierNode{
		ctx:    ctx,
		config: config,

		rabbit: rabbit,

		crawlerRepository: crawlerRepository,
		websiteRepository: websiteRepository,

		blacklists:  blacklists,
		robotParser: &robotParser,
		holter:      holter,
		planner:     planner.NewPlanner[string, string](),
		filter:      filter,
	}

	holter.OnChange(frontier.handleNodeStatusChanges)
	holter.OnTimeout(frontier.handleNodeStatusChanges)

	return frontier
}

func (m *FrontierNode) SetupHolter(mux *http.ServeMux) {
	m.holter.Run(mux)
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

	origin := uri.OriginKey(*url)
	normalizedURI := uri.Normalize(*url)
	siteKey := uri.SiteKey(*url)

	m.websiteRepository.AddDomain(m.ctx, origin)

	if m.blacklists.Contains(siteKey) {
		return nil
	}

	if m.filter.Contains(normalizedURI) {
		return nil
	}

	rbs, err := m.robotParser.Parse(*url)

	if errors.Is(err, robots.ErrNotAllowed) {
		m.blacklists.Add(siteKey)
		return nil
	} else if err != nil {
		return nil
	}

	if !rbs.IsAllow(m.config.UserAgent, *payload.URI) {
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
		Payload: events.URIPayload{
			URI:       &normalizedURI,
			Timestamp: &now,
		},
	}

	log.Printf("Publishing to: %s", key)
	err = broker.Publish(key, event)

	if err != nil {
		return err
	}

	return nil
}

func (m FrontierNode) handleConfirmEvent(payload events.ConfirmPayload) error {
	url, err := url.Parse(*payload.URI)

	if err != nil {
		return nil
	}

	uri := uri.Normalize(*url)
	m.filter.Add(uri)
	return nil
}

func (m FrontierNode) Handle(data []byte) error {
	log.Println("Event Incomming")
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

	switch event.Type {
	case events.FrontierURIRegister:
		var payload events.URIPayload

		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			log.Println(err)
			return nil
		}

		return m.handleURIRegister(payload)
	case events.FrontierCrawlConfirm:
		var payload events.ConfirmPayload

		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			log.Println(err)
			return nil
		}

		return m.handleConfirmEvent(payload)
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
