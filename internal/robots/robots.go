package robots

import (
	"context"
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
	context context.Context

	repo    repository.WebsiteRepository
	fetcher *fetch.Fetcher
	cache   map[string]Robots
}

func NewRobotParser(context context.Context, repo repository.WebsiteRepository, fetcher *fetch.Fetcher) RobotParser {
	return RobotParser{
		context: context,
		repo:    repo,
		fetcher: fetcher,
		cache:   make(map[string]Robots),
	}
}
