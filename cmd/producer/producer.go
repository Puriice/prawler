package main

import (
	"log"
	"time"

	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/enum/hosts"
	"github.com/purrice/prawler/internal/model"
)

func main() {
	seeds, err := config.LoadSeeds()

	if err != nil {
		log.Fatal(err)
	}

	env.Init()
	config := config.GetConfig()
	amqpURL := env.Get("amqp_url", "amqp://guest:guest@localhost:5672")

	rabbitMQ, err := messaging.NewRabbitMQ(amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	defer rabbitMQ.Shutdown()

	broker, err := rabbitMQ.NewBroker(config.ExchangeName)

	if err != nil {
		log.Fatal(err)
	}

	eventType := hosts.HostProduced

	for _, seed := range seeds {
		now := time.Now()
		event := model.HostEvent{
			EventType: &eventType,
			Payload: &model.EventPayload{
				Host:      &seed,
				Timestamp: &now,
			},
		}
		err := broker.Publish("pcrawler.seeds", event)

		if err != nil {
			log.Println(err)
		}
	}
}
