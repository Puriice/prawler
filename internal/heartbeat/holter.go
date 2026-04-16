package heartbeat

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/purrice/prawler/internal/repository"
)

type Node struct {
	UUID     string    `json:"uuid"`
	Status   Status    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
	Payload  any
}

type Holter struct {
	ctx context.Context

	MonitorPeriod time.Duration
	timeout       time.Duration
	grace         time.Duration

	nodes map[string]*Node
	mu    sync.Mutex

	repo repository.CrawlerRepository

	queue chan Event

	onChangeHandler  [2]func(Node)
	onTimeoutHandler [2]func(Node)
}

func NewHolter(
	ctx context.Context,
	timeout time.Duration,
	gracePeriod time.Duration,
	monitorPeriod time.Duration,
	repository repository.CrawlerRepository,

) *Holter {
	holter := &Holter{
		ctx:           ctx,
		MonitorPeriod: monitorPeriod,
		timeout:       timeout,
		grace:         timeout + gracePeriod,
		nodes:         make(map[string]*Node),
		repo:          repository,
		queue:         make(chan Event, 1000),
	}

	holter.onChangeHandler[0] = holter.updateCrawlerStatus
	holter.onTimeoutHandler[0] = holter.removeCrawler

	return holter
}

func (h *Holter) Load(fn func() map[string]*Node) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nodes = fn()
}

func (h *Holter) updateCrawlerStatus(node Node) {
	err := h.repo.UpdateCrawlerStatus(h.ctx, node.UUID, string(node.Status), node.LastSeen)

	if err != nil {
		log.Println(err)
	}
}

func (h *Holter) removeCrawler(node Node) {
	err := h.repo.RemoveCrawler(h.ctx, node.UUID)

	if err != nil {
		log.Println(err)
	}
}

func (h *Holter) registerNode(uuid string) *Node {
	node := &Node{
		UUID:     uuid,
		Status:   Alive,
		LastSeen: time.Now(),
	}

	h.nodes[uuid] = node

	return node
}

func (h *Holter) Get(uuid string) (any, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	nodes, exist := h.nodes[uuid]

	if !exist {
		return nil, false
	}

	if nodes.Status != Alive {
		return nil, false
	}

	return nodes.Payload, nodes.Payload != nil
}

func (h *Holter) Set(uuid string, payload any) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	nodes, exist := h.nodes[uuid]

	if !exist {
		return false
	}

	if nodes.Status != Alive {
		return false
	}

	nodes.Payload = payload
	return true
}

func (h *Holter) Run(mux *http.ServeMux) {
	go h.Monitor()
	h.startEventWorker()

	mux.HandleFunc("POST /api/v1/heartbeat", h.handleBeats)
	mux.HandleFunc("GET /api/v1/nodes", h.handleList)
	mux.HandleFunc("GET /api/v1/heart", h.handleInit)
	mux.HandleFunc("GET /dashboard", h.handleDashboard)
}
