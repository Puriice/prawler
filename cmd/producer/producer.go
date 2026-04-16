package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/master"
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

	broker, err := rabbitMQ.NewBroker(config.ExchangeName.Master)

	if err != nil {
		log.Fatal(err)
	}

	for _, seed := range seeds {
		now := time.Now()

		payload := model.URIPayload{
			URI:       &seed,
			Timestamp: &now,
		}

		bytes, err := json.Marshal(payload)

		if err != nil {
			log.Println(err)
			continue
		}

		event := master.Event{
			Type:    master.EventURIRegister,
			Payload: bytes,
		}

		err = broker.Publish(fmt.Sprintf("%s.uri", config.ExchangeName.Master), event)

		if err != nil {
			log.Println(err)
		}
	}
}
