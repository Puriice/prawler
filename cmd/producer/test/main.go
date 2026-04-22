package main

import (
	"context"
	"log"

	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/frontier"
)

func main() {
	seeds := []string{
		"http://localhost:8081",
	}

	env.Init()
	amqpURL := env.Get("amqp_url", "amqp://guest:guest@localhost:5672")

	rabbitMQ, err := messaging.NewRabbitMQ(context.Background(), amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	defer rabbitMQ.Shutdown()

	client, err := frontier.NewClient(context.Background(), rabbitMQ, nil)

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
