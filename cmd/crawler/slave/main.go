package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/puriice/golibs/pkg/db"
	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/crawler"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/repository"
)

func main() {
	crawlerUUID := uuid.New()

	log.Printf("Crawler UUID: %s\n", crawlerUUID.String())

	env.Init()
	cfg := config.GetConfig()
	amqpURL := env.Get("amqp_url", "amqp://guest:guest@localhost/")

	rabbitMQ, err := messaging.NewRabbitMQ(amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	defer rabbitMQ.Shutdown()

	hostBroker, err := rabbitMQ.NewBroker(cfg.ExchangeName.Hosts)

	if err != nil {
		log.Fatal(err)
	}

	hostListenerConfig := messaging.NewRabbitListenerConfig(cfg.QueueName, "prawler.seeds")
	hostListener, err := hostBroker.NewListenerWithConfig(hostListenerConfig)

	if err != nil {
		log.Fatal(err)
	}

	db, err := db.NewDatabase()

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	webRecordsRepository := repository.NewPostgresWebRecordRepository(db)

	fetcher := fetch.NewFecter(nil)

	crawler := crawler.NewCrawler(cfg.UserAgent, webRecordsRepository, &fetcher)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("Start listening to hosts producing events.")
	if err := hostListener.Subscribe(ctx, crawler.Handle); err != nil {
		log.Println(err)
	}

	log.Println("Shuting down")
}
