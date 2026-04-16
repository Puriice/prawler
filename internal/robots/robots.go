package robots

import (
	"net/url"
	"time"

	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/repository"
)

type Robots struct {
	Host      url.URL
	Raw       *string
	Sitemap   []string
	Timestamp time.Time
}

type RobotParser struct {
	repo    repository.RobotsRepository
	fetcher *fetch.Fetcher
	cache   map[string]Robots
}

func NewRobotParser(repo repository.RobotsRepository, fetcher *fetch.Fetcher) RobotParser {
	return RobotParser{
		repo:    repo,
		fetcher: fetcher,
		cache:   make(map[string]Robots),
	}
}
