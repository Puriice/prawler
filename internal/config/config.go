package config

import (
	"flag"

	"github.com/purrice/prawler/internal/file"
)

var config *Config

type Config struct {
	UserAgent         string `json:"user_agent"`
	ExchangeName      string `json:"exchange_name"`
	QueueName         string `json:"queue_name"`
	CrawlingDelayInMS int    `json:"crawling_delay_ms"`
}

func Default() *Config {
	return &Config{
		UserAgent:         "prawler/1.0 (+https://github.com/Puriice/prawler; Educational project; Contract: purinutt.amartayavis@g.swu.ac.th)",
		ExchangeName:      "pcrawler.events",
		QueueName:         "pcrawler.seeds",
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
