package robots

import (
	"net/url"

	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/repository"
)

type Robots struct {
	Host    url.URL
	Raw     *string
	Sitemap []string
}

type RobotParser struct {
	repo    repository.RobotsRepository
	fetcher *fetch.Fetcher
}

func NewRobotParser(repo repository.RobotsRepository, fetcher *fetch.Fetcher) RobotParser {
	return RobotParser{
		repo:    repo,
		fetcher: fetcher,
	}
}
