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
	"github.com/purrice/prawler/internal/master"
	"github.com/purrice/prawler/internal/repository"
)

func main() {
	env.Init()
	cfg := config.GetConfig()
	amqpURL := env.Get("amqp_url", "amqp://guest:guest@localhost/")

	rabbitMQ, err := messaging.NewRabbitMQ(amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	defer rabbitMQ.Shutdown()

	broker, err := rabbitMQ.NewBroker(cfg.ExchangeName.Master)

	if err != nil {
		log.Fatal(err)
	}

	listenerConfig := messaging.NewRabbitListenerConfig(cfg.ExchangeName.Master, "prawler.master.*")
	listener, err := broker.NewListenerWithConfig(listenerConfig)

	if err != nil {
		log.Fatal(err)
	}

	db, err := db.NewDatabase()

	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repo := repository.NewPostgresMasterRepository(db)
	master := master.NewMasterNode(repo, ctx)

	log.Println("Start listening to slave producing events.")
	if err := listener.Subscribe(ctx, master.Handle); err != nil {
		log.Println(err)
	}

	log.Println("Shuting down")
}
