package crawler

import (
	"encoding/json"

	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
)

type Crawler struct {
	agent         string
	webRecordRepo repository.WebRecordRepository
}

func NewCrawler(agent string, webRecordRepo repository.WebRecordRepository) Crawler {
	return Crawler{
		agent:         agent,
		webRecordRepo: webRecordRepo,
	}
}

func (c Crawler) Handle(data []byte) error {
	var payload model.SeedEvent

	err := json.Unmarshal(data, &payload)

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

	robots.Parse(url)

	return nil
}
