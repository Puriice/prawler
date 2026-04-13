package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/puriice/golibs/pkg/db"
	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/puriice/golibs/pkg/server"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/master"
	"github.com/purrice/prawler/internal/repository"
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

	db, err := db.NewDatabase()

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	host := env.Get("HOST", "")
	port := env.Get("PORT", "5723")

	server := server.NewServer(host, port, nil)
	mux := http.NewServeMux()

	repo := repository.NewPostgresCrawlerRepository(db)
	holter := heartbeat.NewHolter(ctx, 5*time.Second, 10*time.Second, 2*time.Second, repo)
	holter.Run(mux)

	rabbit, listener := setupListener()
	defer rabbit.Shutdown()

	server.Handler = mux
	server.Start()
	master.Run(ctx, db, mux, *listener)
	log.Println("Shuting down")
}
