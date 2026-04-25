package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/puriice/golibs/pkg/db"
	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/crawler"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/frontier"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/worker"
)

func main() {

	env.Init()
	cfg := config.GetConfig()

	amqpURL := env.Get("AMQP_URL", "amqp://guest:guest@localhost/")
	holterEndpoint, err := url.Parse(env.Get("HOLTER_URL", "http://localhost:5723"))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err != nil {
		log.Fatal(err)
	}

	heart := heartbeat.NewHeart(*holterEndpoint, nil)
	go heart.Beat(ctx)

	rabbitMQ, err := messaging.NewRabbitMQ(ctx, amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	defer rabbitMQ.Shutdown()

	db, err := db.NewDatabase()

	if err != nil {
		rabbitMQ.Shutdown()
		log.Fatal(err)
	}

	defer db.Close()

	website := repository.NewPostgresWebsiteRepository(db)

	httpClient := http.Client{
		Timeout: 3 * time.Second,
	}

	fetcher := fetch.NewFecter(&httpClient)
	client, err := frontier.NewClient(ctx, rabbitMQ, website)

	if err != nil {
		rabbitMQ.Shutdown()
		db.Close()
		log.Fatal(err)
	}

	worker := worker.NewManager(ctx, 5, 10)
	worker.SpawnWorker()

	crawler := crawler.NewCrawler(ctx, heart.UUID(), cfg.CrawlingPolicy.UserAgent, website, fetcher, client, worker)
	crawler.Setup(rabbitMQ)

	if err := crawler.Run(); err != nil {
		log.Println(err)
	}

	log.Println("Shuting down")
}
