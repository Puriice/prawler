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
	"github.com/purrice/prawler/internal/robots"
)

func main() {
	env.Init()
	config := config.GetConfig()
	amqpURL := env.Get("amqp_url", "amqp://guest:guest@localhost/")

	rabbitMQ, err := messaging.NewRabbitMQ(amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	defer rabbitMQ.Shutdown()

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

	webRecordsRepository := repository.NewPostgresWebRecordRepository(db)

	robotsRepository := repository.NewPostgresRobotsRepository(db)
	robotParser := robots.NewRobotParser(robotsRepository)

	crawler := crawler.NewCrawler(robotParser, webRecordsRepository)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// var wg sync.WaitGroup

	if err := listener.Subscribe(ctx, crawler.Handle); err != nil {
		log.Println(err)
	}

	log.Println("Shuting down")
}
