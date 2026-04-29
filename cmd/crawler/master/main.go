package main

import (
	"context"
	"flag"
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
)

func main() {
	env.Init()

	db, err := db.NewDatabase()

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	amqpURL := env.Get("AMQP_URL", "amqp://guest:guest@localhost/")
	rabbit, err := messaging.NewRabbitMQ(ctx, amqpURL)

	if err != nil {
		db.Close()
		log.Fatal(err)
	}
	defer rabbit.Shutdown()

	host := env.Get("HOST", "")
	port := env.Get("PORT", "5723")

	server := server.NewServer(host, port, nil)
	mux := http.NewServeMux()

	crawlerRepository := repository.NewPostgresCrawlerRepository(db)
	websiteRepository := repository.NewPostgresWebsiteRepository(db)

	skipLoad := flag.Bool("skip-parsed", false, "use to skip loading parsed page")
	flag.Parse()

	filter := NewBloom(10_000_000, 0.0001)
	filter2 := NewBloom(10_000_000, 0.001)
	frontier := frontier.NewFrontierNode(ctx, rabbit, filter, filter2)
	frontier.Setup(
		crawlerRepository,
		websiteRepository,
		mux,
		!*skipLoad,
	)

	server.Handler = mux
	server.Start()

	frontier.Run()
	<-ctx.Done()
	log.Println("Shuting down")
}
