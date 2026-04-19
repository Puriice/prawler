package main

import (
	"log"

	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/frontier"
)

func main() {
	seeds, err := config.LoadSeeds()

	if err != nil {
		log.Fatal(err)
	}

	env.Init()
	amqpURL := env.Get("amqp_url", "amqp://guest:guest@localhost:5672")

	rabbitMQ, err := messaging.NewRabbitMQ(amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	defer rabbitMQ.Shutdown()

	client, err := frontier.NewClient(rabbitMQ)

	if err != nil {
		log.Fatal(err)
	}

	for _, seed := range seeds {
		err := client.Register(seed, 0, nil)

		if err != nil {
			log.Println(err)
			continue
		}
	}
}
