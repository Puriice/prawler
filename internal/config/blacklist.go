package config

import (
	"context"
	"encoding/json"
	"flag"
	"sync"

	"github.com/purrice/prawler/internal/enum/hosts"
	"github.com/purrice/prawler/internal/file"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/repository"
)

type Blacklists struct {
	mu   sync.RWMutex
	sets map[string]struct{}
	repo repository.BlacklistRepository
}

var (
	once      sync.Once
	blacklist Blacklists
)

func initBlacklist(repo repository.BlacklistRepository) {
	blacklistPath := flag.String("blacklist", "./blacklists.json", "Path for blacklists file")
	flag.Parse()

	var fromJson []string

	err := file.LoadJson(*blacklistPath, &fromJson)
	fromDatabase := repo.Query(context.Background())

	if err != nil {
		fromJson = []string{}
	}

	maxNElement := max(len(fromJson), len(fromDatabase)) // Assume one is subset of another
	sets := make(map[string]struct{}, int(float32(maxNElement)*1.25))

	for _, url := range fromJson {
		sets[url] = struct{}{}
	}

	for _, url := range fromDatabase {
		sets[url] = struct{}{}
	}

	blacklist = Blacklists{
		mu:   sync.RWMutex{},
		sets: sets,
		repo: repo,
	}
}

func NewBlacklist(repo repository.BlacklistRepository) *Blacklists {
	once.Do(func() {
		initBlacklist(repo)
	})

	return &blacklist
}

func (b *Blacklists) Add(urls ...string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, url := range urls {
		b.sets[url] = struct{}{}
	}
}

func (b *Blacklists) Contains(url string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	_, ok := b.sets[url]

	return ok
}

func (b *Blacklists) Remove(url string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.sets, url)
}

func (b *Blacklists) Handle(data []byte) error {
	var payload model.HostEvent

	err := json.Unmarshal(data, &payload)

	if err != nil {
		return err
	}

	if err := payload.IsValid(); err != nil {
		return err
	}

	switch *payload.EventType {
	case hosts.HostBlacklistAdd:
		b.Add(*payload.Payload.Host)
	case hosts.HostBlacklistRemove:
		b.Remove(*payload.Payload.Host)
	}
	return nil
}
