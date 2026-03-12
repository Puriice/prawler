package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/puriice/golibs/pkg/db"
	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/crawler"
	"github.com/purrice/prawler/internal/repository"
)

func main() {
	env.Init()
	config := config.GetConfig()
	amqpURL := env.Get("amqp_url", "amqp://localhost:5672")

	rabbitMQ, err := messaging.NewRabbitMQ(amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	broker, err := rabbitMQ.NewBroker(config.ExchangeName)

	if err != nil {
		log.Fatal(err)
	}

	listenerConfig := messaging.NewRabbitListenerConfig(config.QueueName, "pcrawler.seeds")
	listener, err := broker.NewListenerWithConfig(listenerConfig)

	if err != nil {
		log.Fatal(err)
	}

	db, err := db.NewDatabase()

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	webRecordsRepository := repository.NewPostgresWebRecordRepository(db)

	crawler := crawler.NewCrawler(config.UserAgent, webRecordsRepository)

	go func() {
		if err := listener.Subscribe(ctx, crawler.Handle); err != nil {
			log.Fatal(err)
		}
	}()

	<-exit

	cancel()
}
