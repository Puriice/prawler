package config

import (
	"encoding/json"
	"flag"
	"os"
)

var config *Config

type Config struct {
	UserAgent    string `json:"user_agent"`
	ExchangeName string `json:"exchange_name"`
	QueueName    string `json:"queue_name"`
}

func Default() *Config {
	return &Config{
		UserAgent:    "pcrawler/1.0",
		ExchangeName: "pcrawler.events",
		QueueName:    "pcrawler.seeds",
	}
}

func parseConfig() *Config {
	configPath := flag.String("config", "./config/crawler.json", "path to config file")
	flag.Parse()

	content, err := os.ReadFile(*configPath)

	if err != nil {
		return Default()
	}

	var config Config

	err = json.Unmarshal(content, &config)

	if err != nil {
		return Default()
	}

	return &config
}

func GetConfig() *Config {
	if config == nil {
		config = parseConfig()
	}

	return config
}
