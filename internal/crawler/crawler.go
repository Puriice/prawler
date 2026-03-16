package crawler

import (
	"encoding/json"
	"log"

	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/enum/hosts"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
)

type Crawler struct {
	robots        robots.RobotParser
	webRecordRepo repository.WebRecordRepository
	blacklist     *config.Blacklists
}

func NewCrawler(
	robots robots.RobotParser,
	webRecordRepo repository.WebRecordRepository,
	blacklists *config.Blacklists,
) Crawler {
	return Crawler{
		robots:        robots,
		webRecordRepo: webRecordRepo,
		blacklist:     blacklists,
	}
}

func (c Crawler) handleProduceEvent(payload model.EventPayload) error {
	url, err := payload.GetHost()

	if err != nil {
		return err
	}

	rbs, err := c.robots.Parse(*url)

	if err != nil {
		return err
	}

	log.Printf("%s/robots.txt:", url.Host)
	// log.Println(*robots.Raw)
	robots.Println(*rbs)

	return nil
}

func (c Crawler) handleBlacklistEvent(payload model.EventPayload) error {
	c.blacklist.Add(*payload.Host)
	return nil
}

func (c Crawler) Handle(data []byte) error {
	var payload model.HostEvent

	err := json.Unmarshal(data, &payload)

	if err != nil {
		return err
	}

	if err := payload.IsValid(); err != nil {
		return err
	}

	switch *payload.EventType {
	case hosts.HostProduced:
		return c.handleProduceEvent(*payload.Payload)
	case hosts.HostBlacklist:
		return c.handleBlacklistEvent(*payload.Payload)

	}
	return nil
}
