package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/puriice/golibs/pkg/db"
	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/puriice/golibs/pkg/server"
	"github.com/purrice/prawler/internal/frontier"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/set"
)

func main() {
	env.Init()

	db, err := db.NewDatabase()

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	amqpURL := env.Get("AMQP_URL", "amqp://guest:guest@localhost/")
	rabbit, err := messaging.NewRabbitMQ(amqpURL)

	if err != nil {
		db.Close()
		log.Fatal(err)
	}
	defer rabbit.Shutdown()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	host := env.Get("HOST", "")
	port := env.Get("PORT", "5723")

	server := server.NewServer(host, port, nil)
	mux := http.NewServeMux()

	crawlerRepository := repository.NewPostgresCrawlerRepository(db)
	websiteRepository := repository.NewPostgresWebsiteRepository(db)

	filter := set.NewSet[string]()
	frontier := frontier.NewFrontierNode(ctx, rabbit, filter)
	frontier.Setup(
		crawlerRepository,
		websiteRepository,
		mux,
	)

	server.Handler = mux
	server.Start()

	frontier.Run()
	log.Println("Shuting down")
}
