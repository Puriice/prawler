package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

type migrator struct {
	config *config.Config
	broker *messaging.RabbitBroker
	mu     sync.RWMutex
}

func (m *migrator) handleMigration(data []byte) error {
	var event events.FrontierEvent

	err := json.Unmarshal(data, &event)

	if err != nil {
		log.Println(err)
		return err
	}

	if err := event.IsValid(); err != nil {
		log.Println(err)
		return err
	}
	log.Printf("Event Incomming: %s start migrating", event.Type)

	m.mu.RLock()
	Register := fmt.Sprintf("%s.uri", m.config.ExchangeName.Frontier)
	Confirm := fmt.Sprintf("%s.confirm", m.config.ExchangeName.Frontier)
	Backoff := fmt.Sprintf("%s.backoff", m.config.ExchangeName.Frontier)

	switch event.Type {
	case events.FrontierURIRegister:
		m.broker.PublishRaw(
			Register,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        data,
			},
		)
	case events.FrontierCrawlConfirm:
		m.broker.PublishRaw(
			Confirm,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        data,
			},
		)
	case events.FrontierBackoff:
		m.broker.PublishRaw(
			Backoff,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        data,
			},
		)
	}
	m.mu.RUnlock()

	return nil
}
