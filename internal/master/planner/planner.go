package planner

import (
	"net/url"
	"sync"

	"github.com/purrice/prawler/internal/multikey"
	"github.com/purrice/prawler/internal/origin"
)

type Planner struct {
	register *multikey.Map[url.URL, string]
	mu       sync.Mutex
}

func NewPlanner() *Planner {
	return &Planner{
		register: multikey.New[url.URL, string](),
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

	origin := origin.GetOrigin(uri)

	crawler, ok := p.register.Get(origin)

	if ok {
		return crawler, true
	}

	crawler, ok = p.register.GetValueWithLeastKey()

	if !ok {
		return "", false
	}

	p.register.Put(origin, crawler)

	return crawler, true
}
