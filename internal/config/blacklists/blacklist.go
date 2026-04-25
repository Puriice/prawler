package blacklists

import (
	"context"
	"encoding/json"
	"flag"
	"net/url"
	"sync"

	"github.com/purrice/prawler/internal/events"
	"github.com/purrice/prawler/internal/file"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/uri"
)

type Blacklists struct {
	mu   sync.RWMutex
	sets map[string]struct{}
	repo repository.WebsiteRepository
}

var (
	once      sync.Once
	blacklist Blacklists
)

func initBlacklist(repo repository.WebsiteRepository) {
	blacklistPath := flag.String("blacklist", "./blacklists.json", "Path for blacklists file")
	flag.Parse()

	var fromJson []string

	err := file.LoadJson(*blacklistPath, &fromJson)
	fromDatabase := repo.GetBlacklistDomain(context.Background())

	if err != nil {
		fromJson = []string{}
	}

	maxNElement := min(max(len(fromJson), len(fromDatabase)), 4) // Assume one is subset of another
	sets := make(map[string]struct{}, int(float32(maxNElement)*1.25))

	blacklist = Blacklists{
		mu:   sync.RWMutex{},
		sets: sets,
		repo: repo,
	}

	blacklist.Add(fromJson...)
	blacklist.addWithoutSave(fromDatabase...)
}

func NewBlacklist(repo repository.WebsiteRepository) *Blacklists {
	once.Do(func() {
		initBlacklist(repo)
	})

	return &blacklist
}

func (b *Blacklists) addWithoutSave(urls ...string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, u := range urls {
		if url, err := url.Parse(u); err == nil {
			sitekey := uri.StickySessionKey(*url)

			b.sets[sitekey] = struct{}{}
		}
	}
}

func (b *Blacklists) Add(urls ...string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, u := range urls {
		if url, err := url.Parse(u); err == nil {
			origin := uri.OriginKey(*url)
			sitekey := uri.StickySessionKey(*url)

			b.sets[sitekey] = struct{}{}
			b.repo.BlacklistDomain(context.Background(), *origin)
		}
	}
}

func (b *Blacklists) Contains(u string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	url, err := url.Parse(u)

	if err != nil {
		return false
	}

	sitekey := uri.StickySessionKey(*url)

	_, ok := b.sets[sitekey]

	return ok
}

func (b *Blacklists) Remove(u string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	url, err := url.Parse(u)

	if err != nil {
		return
	}

	origin := uri.OriginKey(*url)
	sitekey := uri.StickySessionKey(*url)

	delete(b.sets, sitekey)
	b.repo.BlacklistDomain(context.Background(), *origin)
}

func (b *Blacklists) Handle(data []byte) error {
	var event events.BlacklistEvent

	err := json.Unmarshal(data, &event)

	if err != nil {
		return err
	}

	if err := event.IsValid(); err != nil {
		return err
	}

	if err := event.Payload.IsValid(); err != nil {
		return err
	}

	switch event.Type {
	case events.BlacklistAdd:
		b.Add(*event.Payload.URI)
	case events.BlacklistRemove:
		b.Remove(*event.Payload.URI)
	}
	return nil
}
