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
	ctx           context.Context
	MonitorPeriod time.Duration
	timeout       time.Duration
	grace         time.Duration
	nodes         map[string]*Node
	mu            sync.Mutex
	repo          repository.CrawlerRepository
}

func NewHolter(
	ctx context.Context,
	timeout time.Duration,
	gracePeriod time.Duration,
	monitorPeriod time.Duration,
	repository repository.CrawlerRepository,

) *Holter {
	return &Holter{
		ctx:           ctx,
		MonitorPeriod: monitorPeriod,
		timeout:       timeout,
		grace:         timeout + gracePeriod,
		nodes:         make(map[string]*Node),
		repo:          repository,
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

func (h *Holter) triggerBeatHandler(node Node) {
	err := h.repo.UpdateCrawlerStatus(h.ctx, node.UUID, string(node.Status), node.LastSeen)

	if err != nil {
		log.Println(err)
	}
}

func (h *Holter) triggerTimeoutHandler(node Node) {
	err := h.repo.RemoveCrawler(h.ctx, node.UUID)
	if err != nil {
		log.Println(err)
	}
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

	mux.HandleFunc("POST /api/v1/heartbeat", h.handleBeats)
	mux.HandleFunc("GET /api/v1/nodes", h.handleList)
	mux.HandleFunc("GET /api/v1/heart", h.handleInit)
	mux.HandleFunc("GET /dashboard", h.handleDashboard)
}
