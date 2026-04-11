package config

import (
	"flag"
	"fmt"

	"github.com/purrice/prawler/internal/file"
)

var config *Config

type Contact struct {
	Email string `json:"email"`
	Git   string `json:"git"`
}

type ExchangeConfig struct {
	Hosts      string `json:"hosts"`
	Blacklists string `json:"blacklists"`
	Master     string `json:"master"`
}

type Config struct {
	Version           float32        `json:"version"`
	Contact           Contact        `json:"contact"`
	UserAgent         string         `json:"user_agent"`
	ExchangeName      ExchangeConfig `json:"exchange_name"`
	QueueName         string         `json:"queue_name"`
	CrawlingDelayInMS int            `json:"crawling_delay_ms"`
}

func Default() *Config {
	return &Config{
		Version: 1.0,
		Contact: Contact{
			Email: "purinutt.amartayavis@g.swu.ac.th",
			Git:   "https://github.com/Puriice/prawler",
		},
		UserAgent: "prawler",
		ExchangeName: ExchangeConfig{
			Hosts:      "prawler.hosts",
			Blacklists: "prawler.blacklists",
			Master:     "prawler.master",
		},
		QueueName:         "pcrawler.hosts",
		CrawlingDelayInMS: 1000,
	}
}

func parseConfig() *Config {
	configPath := flag.String("config", "./config.json", "path to config file")
	flag.Parse()

	var config = Default()

	err := file.LoadJson(*configPath, &config)

	if err != nil {
		return Default()
	}

	return config
}

func GetConfig() *Config {
	if config == nil {
		config = parseConfig()
	}

	return config
}

func (c Config) GetDisplayUserAgent() string {
	return fmt.Sprintf("%s/%f (%s; Email: %s)", c.UserAgent, c.Version, c.Contact.Git, c.Contact.Email)
}
