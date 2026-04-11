package crawler

import (
	"encoding/json"

	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/enum/hosts"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/origin"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
	"github.com/purrice/prawler/internal/types"
)

type Crawler struct {
	agent         string
	robots        robots.RobotParser
	webRecordRepo repository.WebRecordRepository
	blacklist     *config.Blacklists
}

func NewCrawler(
	userAgent string,
	robots robots.RobotParser,
	webRecordRepo repository.WebRecordRepository,
	blacklists *config.Blacklists,
) Crawler {
	return Crawler{
		agent:         userAgent,
		robots:        robots,
		webRecordRepo: webRecordRepo,
		blacklist:     blacklists,
	}
}

func (c Crawler) handleProduceEvent(payload model.HostPayload) error {
	url, err := payload.GetHost()

	if err != nil {
		return err
	}

	origin := origin.GetOrigin(*url)

	if c.blacklist.Contains(origin.String()) {
		return nil // the target is on blacklisted act as consumed
	}

	rbs, err := c.robots.Parse(*url)

	if err != nil {
		return err
	}

	if !rbs.IsAllow(c.agent, url.String()) {
		return nil // the target is not allow to crawl act as consumed
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

	payload, ok := event.Payload.(*model.HostPayload)

	if !ok {
		return types.ErrInvalidPaylod
	}

	switch *event.EventType {
	case hosts.HostProduced:
		return c.handleProduceEvent(*payload)

	}
	return nil
}
