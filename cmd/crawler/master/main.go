package main

import (
	"log"
	"time"

	"github.com/puriice/golibs/pkg/db"
	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/master"
)

func setupListener() (*messaging.RabbitMQ, *messaging.RabbitListener) {
	cfg := config.GetConfig()
	amqpURL := env.Get("amqp_url", "amqp://guest:guest@localhost/")

	rabbitMQ, err := messaging.NewRabbitMQ(amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	broker, err := rabbitMQ.NewBroker(cfg.ExchangeName.Master)

	if err != nil {
		log.Fatal(err)
	}

	listenerConfig := messaging.NewRabbitListenerConfig(cfg.ExchangeName.Master, "prawler.master.*")
	listener, err := broker.NewListenerWithConfig(listenerConfig)

	return rabbitMQ, listener
}

func main() {
	env.Init()

	holter := heartbeat.NewHolter(5*time.Second, 5*time.Second, 2*time.Second)
	holter.Run()

	rabbit, listener := setupListener()
	defer rabbit.Shutdown()

	db, err := db.NewDatabase()

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	master.Run(db, *listener)
	log.Println("Shuting down")
}
