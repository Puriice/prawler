package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/puriice/golibs/pkg/db"
	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/master"
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

	holterHandler := NewHolterHandler(ctx, db)
	holter := heartbeat.NewHolter(5*time.Second, 5*time.Second, 2*time.Second)
	holter.OnBeats(holterHandler.handleOnBeat)
	holter.OnTimeout(holterHandler.handleOnTimeout)
	holter.Run()

	rabbit, listener := setupListener()
	defer rabbit.Shutdown()

	master.Run(ctx, db, *listener)
	log.Println("Shuting down")
}
