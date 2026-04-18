package backoff

import (
	"math/rand"
	"sync"
	"time"
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
	cfg    Config
}

func Default() Config {
	return Config{
		BaseDelay: 3 * time.Second,
		MaxDelay:  60 * time.Second,
		Jitter:    0.5,

		Multiplier429: 2,
		Multiplier503: 3,
		Multiplier5xx: 2.5,
	}
}

func NewManager(cfg Config) *Manager {
	return &Manager{
		states: make(map[string]*state),
		cfg:    cfg,
	}
}

func (m *Manager) Add(sitekey string, httpStatus int, retryAfter time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.states[sitekey]
	if !ok {
		s = &state{}
		m.states[sitekey] = s
	}

	now := time.Now()

	// Calculate delay
	var delay time.Duration

	if retryAfter > 0 {
		delay = max(retryAfter)
	} else {
		delay = m.cfg.BaseDelay
		delay *= (1 << s.attempt)
		delay = min(delay, m.cfg.MaxDelay)
	}

	switch {
	case httpStatus == 429:
		delay *= time.Duration(m.cfg.Multiplier429)
	case httpStatus == 503:
		delay *= time.Duration(m.cfg.Multiplier503)
	case httpStatus >= 500:
		delay *= time.Duration(m.cfg.Multiplier5xx)
	}

	// Apply jitter
	jitter := time.Duration(rand.Float64() * m.cfg.Jitter * float64(delay))
	delay = delay + jitter

	// Update state
	s.attempt++
	s.next = now.Add(delay)
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
