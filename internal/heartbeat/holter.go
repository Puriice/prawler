package heartbeat

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/purrice/prawler/internal/repository"
)

type Handler func(node Node)

type handlers struct {
	onActivate Handler
	onBeats    Handler
	onTimeout  Handler
}

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
	handlers      handlers
	repo          repository.CrawlerRepository
}

func noAction(node Node) {}

func NewHolter(timeout time.Duration, gracePeriod time.Duration, monitorPeriod time.Duration) *Holter {
	return &Holter{
		MonitorPeriod: monitorPeriod,
		timeout:       timeout,
		grace:         timeout + gracePeriod,
		nodes:         make(map[string]*Node),
		handlers: handlers{
			onActivate: noAction,
			onBeats:    noAction,
			onTimeout:  noAction,
		},
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

func (h *Holter) OnActivate(handler Handler) {
	h.handlers.onActivate = handler
}

func (h *Holter) OnBeats(handler Handler) {
	h.handlers.onBeats = handler
}

func (h *Holter) OnTimeout(handler Handler) {
	h.handlers.onTimeout = handler
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
