package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/puriice/golibs/pkg/db"
	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/puriice/golibs/pkg/middleware"
	"github.com/puriice/golibs/pkg/server"
	"github.com/purrice/prawler/internal/master"
	"github.com/purrice/prawler/internal/set"
)

func main() {
	env.Init()

	db, err := db.NewDatabase()

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	amqpURL := env.Get("amqp_url", "amqp://guest:guest@localhost/")
	rabbit, err := messaging.NewRabbitMQ(amqpURL)

	if err != nil {
		log.Fatal(err)
	}
	defer rabbit.Shutdown()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	host := env.Get("HOST", "")
	port := env.Get("PORT", "5723")

	server := server.NewServer(host, port, nil)
	mux := http.NewServeMux()

	filter := set.NewSet[url.URL]()
	master := master.NewMasterNode(ctx, db, rabbit, &filter)
	master.SetupHolter(mux)

	server.Handler = middleware.Logger(mux)
	server.Start()

	master.Run()
	log.Println("Shuting down")
}
