package planner

import (
	"net/url"
	"sync"

	"github.com/purrice/prawler/internal/key"
	"github.com/purrice/prawler/internal/multikey"
)

type Planner struct {
	register *multikey.Map[string, string]
	mu       sync.Mutex
}

func NewPlanner() *Planner {
	return &Planner{
		register: multikey.New[string, string](),
	}
}

func (p *Planner) AddCrawler(uuid string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.register.AddValue(uuid)
}

func (p *Planner) RemoveCrawler(uuid string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.register.RemoveByValue(uuid)
}

func (p *Planner) Plan(uri url.URL) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := key.SiteKey(uri)

	crawler, ok := p.register.Get(key)

	if ok {
		return crawler, true
	}

	crawler, ok = p.register.GetValueWithLeastKey()

	if !ok {
		return "", false
	}

	p.register.Put(key, crawler)

	return crawler, true
}
