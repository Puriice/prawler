package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/puriice/golibs/pkg/db"
	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/crawler"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/frontier"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/repository"
)

func main() {

	env.Init()
	cfg := config.GetConfig()

	amqpURL := env.Get("amqp_url", "amqp://guest:guest@localhost/")
	holterEndpoint, err := url.Parse(env.Get("holter_url", "http://localhost:5723"))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err != nil {
		log.Fatal(err)
	}

	heart := heartbeat.NewHeart(*holterEndpoint, nil)
	go heart.Beat(ctx)

	rabbitMQ, err := messaging.NewRabbitMQ(amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	defer rabbitMQ.Shutdown()

	hostBroker, err := rabbitMQ.NewBroker(cfg.ExchangeName.URI)

	if err != nil {
		rabbitMQ.Shutdown()
		log.Fatal(err)
	}

	qName := fmt.Sprintf("%s.%s", cfg.QueueName, heart.UUID())
	log.Printf("Listening to: %s", qName)
	hostListenerConfig := messaging.NewRabbitListenerConfig(qName, qName)
	hostListenerConfig.AutoDelete = true
	hostListenerConfig.Durable = true
	hostListenerConfig.Exclusive = true
	hostListener, err := hostBroker.NewListenerWithConfig(hostListenerConfig)

	if err != nil {
		rabbitMQ.Shutdown()
		log.Fatal(err)
	}

	db, err := db.NewDatabase()

	if err != nil {
		rabbitMQ.Shutdown()
		log.Fatal(err)
	}

	defer db.Close()

	webRecordsRepository := repository.NewPostgresWebRecordRepository(db)

	fetcher := fetch.NewFecter(nil)
	client, err := frontier.NewClient(rabbitMQ)

	if err != nil {
		rabbitMQ.Shutdown()
		db.Close()
		log.Fatal(err)
	}

	crawler := crawler.NewCrawler(cfg.UserAgent, webRecordsRepository, &fetcher, client)

	log.Println("Start listening to hosts producing events.")
	if err := hostListener.Subscribe(ctx, crawler.Handle); err != nil {
		log.Println(err)
	}

	log.Println("Shuting down")
}
