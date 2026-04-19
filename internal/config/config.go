package config

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/purrice/prawler/internal/file"
)

var (
	config     *Config
	configOnce sync.Once
)

type Contact struct {
	Email       string `json:"email"`
	Description string `json:"description"`
	Git         string `json:"git"`
}

type ExchangeConfig struct {
	URI        string `json:"uri"`
	Blacklists string `json:"blacklists"`
	Frontier   string `json:"frontier"`
	Backoff    string `json:"backoff"`
}

type BackoffPolicy struct {
	Jitter        float64 `json:"jitter"`
	Multiplier429 float64 `json:"multiplier_429"`
	Multiplier503 float64 `json:"multiplier_503"`
	Multiplier5XX float64 `json:"multiplier_5XX"`
}

type CrawlingPolicy struct {
	UserAgent                string        `json:"user_agent"`
	Backoff                  BackoffPolicy `json:"backoff"`
	MinimumCrawlingDelayInMS time.Duration `json:"minimum_crawling_delay_ms"`
	MaximumCrawlingDelayInMS time.Duration `json:"maximum_crawling_delay_ms"`
	MaximumCrawlingDepth     int           `json:"maximum_crawling_depth"`
	MaximumCrawlingAttempt   int           `json:"maximum_crawling_attempt"`
}

type Config struct {
	Version        float32        `json:"version"`
	Contact        Contact        `json:"contact"`
	ExchangeName   ExchangeConfig `json:"exchange_name"`
	QueueName      string         `json:"queue_name"`
	CrawlingPolicy CrawlingPolicy `json:"policy"`
}

func Default() *Config {
	return &Config{
		Version: 1.0,
		Contact: Contact{
			Email: "purinutt.amartayavis@g.swu.ac.th",
			Git:   "https://github.com/Puriice/prawler",
		},
		ExchangeName: ExchangeConfig{
			URI:        "prawler.uri",
			Blacklists: "prawler.blacklists",
			Frontier:   "prawler.frontier",
			Backoff:    "prawler.backoff",
		},
		QueueName: "prawler.uri",
		CrawlingPolicy: CrawlingPolicy{
			UserAgent: "prawler",
			Backoff: BackoffPolicy{
				Jitter:        0.5,
				Multiplier429: 2,
				Multiplier503: 3,
				Multiplier5XX: 2.5,
			},
			MinimumCrawlingDelayInMS: time.Duration(3000),
			MaximumCrawlingDelayInMS: time.Duration(60000),
			MaximumCrawlingDepth:     5,
			MaximumCrawlingAttempt:   5,
		},
	}
}

func initConfig() {
	configPath := flag.String("config", "./config.json", "path to config file")
	flag.Parse()

	config = Default()

	err := file.LoadJson(*configPath, &config)

	if err != nil {
		config = Default()
	}

	config.CrawlingPolicy.MinimumCrawlingDelayInMS = config.CrawlingPolicy.MinimumCrawlingDelayInMS * time.Millisecond
	config.CrawlingPolicy.MaximumCrawlingDelayInMS = config.CrawlingPolicy.MaximumCrawlingDelayInMS * time.Millisecond
}

func GetConfig() *Config {
	configOnce.Do(initConfig)

	return config
}

func (c Config) GetDisplayUserAgent() string {
	return fmt.Sprintf("%s/%f (%s; %s Email: %s)", c.CrawlingPolicy.UserAgent, c.Version, c.Contact.Git, c.Contact.Description, c.Contact.Email)
}
