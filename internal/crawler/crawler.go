package crawler

import (
	"encoding/json"
	"log"

	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
)

type Crawler struct {
	robots        robots.RobotParser
	webRecordRepo repository.WebRecordRepository
	blacklist     *config.Blacklists
}

func NewCrawler(robots robots.RobotParser, webRecordRepo repository.WebRecordRepository, blacklists *config.Blacklists) Crawler {
	return Crawler{
		robots:        robots,
		webRecordRepo: webRecordRepo,
		blacklist:     blacklists,
	}
}

func (c Crawler) Handle(data []byte) error {
	var payload model.SeedEvent

	err := json.Unmarshal(data, &payload)

	log.Println(payload)

	if err != nil {
		return err
	}

	if err := payload.IsValid(); err != nil {
		return err
	}

	url, err := payload.GetSeed()

	if err != nil {
		return err
	}

	robots, err := c.robots.Parse(*url)

	if err != nil {
		return err
	}

	log.Printf("%s/robots.txt:", url.Host)
	log.Println(*robots.Raw)

	return nil
}
