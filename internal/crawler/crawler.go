package crawler

import (
	"encoding/json"
	"log"

	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/enum/hosts"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
	"github.com/purrice/prawler/internal/types"
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

func (c Crawler) handleProduceEvent(payload model.HostPayload) error {
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
