package robots

import (
	"time"

	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/repository"
	"golang.org/x/time/rate"
)

type Rule struct {
	Allow    []string
	Disallow []string
}

type Rules map[string]*Rule
type Robots struct {
	Host    string
	Raw     *string
	Rules   Rules
	Sitemap []string
}

type RobotParser struct {
	repo    repository.RobotsRepository
	limiter *rate.Limiter
}

func NewRobotParser(repo repository.RobotsRepository) RobotParser {
	delay := time.Duration(config.GetConfig().CrawlingDelayInMS) * time.Millisecond
	limiter := rate.NewLimiter(rate.Every(delay), 1)

	return RobotParser{
		limiter: limiter,
		repo:    repo,
	}
}
