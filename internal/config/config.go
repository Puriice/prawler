package config

import (
	"flag"

	"github.com/purrice/prawler/internal/file"
)

var config *Config

type ExchangeConfig struct {
	Hosts      string `json:"hosts"`
	Blacklists string `json:"blacklists"`
}

type Config struct {
	UserAgent         string         `json:"user_agent"`
	ExchangeName      ExchangeConfig `json:"exchange_name"`
	QueueName         string         `json:"queue_name"`
	CrawlingDelayInMS int            `json:"crawling_delay_ms"`
}

func Default() *Config {
	return &Config{
		UserAgent: "prawler/1.0 (+https://github.com/Puriice/prawler; Educational project; Contract: purinutt.amartayavis@g.swu.ac.th)",
		ExchangeName: ExchangeConfig{
			Hosts:      "prawler.hosts",
			Blacklists: "prawler.blacklists",
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
