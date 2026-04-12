package crawler

import (
	"encoding/json"

	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/enum/hosts"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/origin"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/types"
)

type Crawler struct {
	agent         string
	webRecordRepo repository.WebRecordRepository
	blacklist     *config.Blacklists
	fetcher       *fetch.Fetcher
}

func NewCrawler(
	userAgent string,
	webRecordRepo repository.WebRecordRepository,
	blacklists *config.Blacklists,
	fetcher *fetch.Fetcher,
) Crawler {
	return Crawler{
		agent:         userAgent,
		webRecordRepo: webRecordRepo,
		blacklist:     blacklists,
		fetcher:       fetcher,
	}
}

func (c Crawler) handleProduceEvent(payload model.URIPayload) error {
	url, err := payload.GetHost()

	if err != nil {
		return err
	}

	origin := origin.GetOrigin(*url)

	if c.blacklist.Contains(origin.String()) {
		return nil // the target is on blacklisted act as consumed
	}

	return nil
}

func (c Crawler) Handle(data []byte) error {
	var event model.Event

	err := json.Unmarshal(data, &event)

	if err != nil {
		return err
	}

	if err := event.IsValid(); err != nil {
		return err
	}

	payload, ok := event.Payload.(*model.URIPayload)

	if !ok {
		return types.ErrInvalidPaylod
	}

	switch *event.EventType {
	case hosts.HostProduced:
		return c.handleProduceEvent(*payload)

	}
	return nil
}
