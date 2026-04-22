package backoff

import (
	"math/rand"
	"sync"
	"time"

	"github.com/purrice/prawler/internal/config"
)

type Config struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
	Jitter    float64 // 0.0 - 1.0 (e.g. 0.5 = ±50%)

	Multiplier429 float64
	Multiplier503 float64
	Multiplier5xx float64
}

type state struct {
	attempt int
	next    time.Time
}

type Manager struct {
	mu     sync.Mutex
	states map[string]*state
	cfg    *config.Config
}

func NewManager() *Manager {
	return &Manager{
		states: make(map[string]*state),
		cfg:    config.GetConfig(),
	}
}

func (m *Manager) Add(sitekey string, httpStatus int, retryAfter time.Duration) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.states[sitekey]
	if !ok {
		s = &state{}
		m.states[sitekey] = s
	}

	now := time.Now()

	// Calculate delay
	delay := max(retryAfter, m.cfg.CrawlingPolicy.MinimumCrawlingDelayInMS)
	delay *= (1 << s.attempt)

	switch {
	case httpStatus == 429:
		delay *= time.Duration(m.cfg.CrawlingPolicy.Backoff.Multiplier429)
	case httpStatus == 503:
		delay *= time.Duration(m.cfg.CrawlingPolicy.Backoff.Multiplier503)
	case httpStatus >= 500:
		delay *= time.Duration(m.cfg.CrawlingPolicy.Backoff.Multiplier5XX)
	}

	delay = min(delay, m.cfg.CrawlingPolicy.MaximumCrawlingDelayInMS)

	// Apply jitter
	jitter := time.Duration(rand.Float64() * m.cfg.CrawlingPolicy.Backoff.Jitter * float64(delay))
	delay = delay + jitter

	// Update state
	s.attempt++
	s.next = now.Add(delay)

	return time.Until(s.next)
}

func (m *Manager) Set(sitekey string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.states[sitekey]
	if !ok {
		s = &state{}
		m.states[sitekey] = s
	}

	s.next = time.Now().Add(delay)
}

func (m *Manager) Attempt(sitekey string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.states[sitekey]

	if !ok {
		return 0
	}

	return s.attempt
}

// Get how long to wait before next request
func (m *Manager) NextDelay(sitekey string) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.states[sitekey]

	if !ok {
		return 0
	}

	now := time.Now()

	if now.After(s.next) {
		return time.Duration(0)
	}

	return time.Until(s.next)
}

// Call when request succeeds → reset backoff
func (m *Manager) Reset(sitekey string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, sitekey)
}
