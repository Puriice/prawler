package master

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
	"github.com/purrice/prawler/internal/crawler"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/master/planner"
	"github.com/purrice/prawler/internal/model"
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
type MasterNode struct {
	ctx    context.Context
	config *config.Config

	rabbit *messaging.RabbitMQ

	masterRepository  repository.MasterRepository
	crawlerRepository repository.CrawlerRepository

	blacklists  *blacklists.Blacklists
	robotParser *robots.RobotParser
	holter      *heartbeat.Holter
	planner     *planner.Planner
	filter      Filter
}

func NewMasterNode(
	ctx context.Context,
	db *pgxpool.Pool,
	rabbit *messaging.RabbitMQ,
	filter Filter,
) MasterNode {
	fetcher := fetch.NewFecter(nil)

	repo := repository.NewPostgresMasterRepository(db)
	crawlerRepo := repository.NewPostgresCrawlerRepository(db)
	robotsRepository := repository.NewPostgresRobotsRepository(db)
	blacklistRepo := repository.NewPostgresBlacklistRepository(db)

	robotParser := robots.NewRobotParser(robotsRepository, &fetcher)
	blacklists := blacklists.NewBlacklist(blacklistRepo)
	holter := heartbeat.NewHolter(ctx, 5*time.Second, 10*time.Second, 2*time.Second, crawlerRepo)

	config := config.GetConfig()

	master := MasterNode{
		ctx:    ctx,
		config: config,

		rabbit: rabbit,

		masterRepository:  repo,
		crawlerRepository: crawlerRepo,

		blacklists:  blacklists,
		robotParser: &robotParser,
		holter:      holter,
		planner:     planner.NewPlanner(),
		filter:      filter,
	}

	holter.OnChange(master.handleNodeStatusChanges)
	holter.OnTimeout(master.handleNodeStatusChanges)

	return master
}

func (m *MasterNode) SetupHolter(mux *http.ServeMux) {
	m.holter.Run(mux)
}

func (m *MasterNode) handleNodeStatusChanges(node heartbeat.Node) {
	switch node.Status {
	case heartbeat.Alive:
		log.Printf("Add %s to the planner.", node.UUID)
		m.planner.AddCrawler(node.UUID)
	case heartbeat.Unconscious, heartbeat.Dead:
		log.Printf("Remove %s from the planner.", node.UUID)
		m.planner.RemoveCrawler(node.UUID)
	}
}

func (m MasterNode) handleURIRegister(payload model.URIPayload) error {
	url, err := url.Parse(*payload.URI)
	log.Printf("RECEIVED: %s\n", *payload.URI)

	if err != nil {
		return nil // Error parsing url return nil because we don't want a retry
	}

	normalizedURI := uri.Normalize(*url)
	siteKey := uri.SiteKey(*url)

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

	crawlerUUID, ok := m.planner.Plan(*url)

	if !ok {
		return ErrNoAvaliableCrawler
	}

	broker, err := m.rabbit.NewBroker(m.config.ExchangeName.URI)

	if err != nil {
		return err
	}

	now := time.Now()

	key := fmt.Sprintf("%s.%s", m.config.QueueName, crawlerUUID)

	event := crawler.Event{
		Type: crawler.EventURI,
		Payload: model.URIPayload{
			URI:       payload.URI,
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

func (m MasterNode) handleConfirmEvent(payload model.URIPayload) error {
	url, err := url.Parse(*payload.URI)

	if err != nil {
		return nil
	}

	uri := uri.Normalize(*url)
	m.filter.Add(uri)
	return nil
}

func (m MasterNode) Handle(data []byte) error {
	log.Println("Event Incomming")
	var event Event

	err := json.Unmarshal(data, &event)

	if err != nil {
		log.Println(err)
		return err
	}

	if err := event.IsValid(); err != nil {
		log.Println(err)
		return err
	}

	if err := event.Payload.IsValid(); err != nil {
		log.Println(err)
		return nil
	}

	switch event.Type {
	case EventURIRegister:
		return m.handleURIRegister(event.Payload)
	case EventCrawlConfirm:
		return m.handleConfirmEvent(event.Payload)
	}

	return nil
}

func (m MasterNode) Run() {
	broker, err := m.rabbit.NewBroker(m.config.ExchangeName.Master)

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

	listenerConfig := messaging.NewRabbitListenerConfig(m.config.ExchangeName.Master, fmt.Sprintf("%s.#", m.config.ExchangeName.Master))
	listener, err := broker.NewListenerWithConfig(listenerConfig)

	log.Println("Start listening to slave producing events.")
	if err := listener.Subscribe(m.ctx, m.Handle); err != nil {
		log.Println(err)
	}
}
