package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/worker"
)

const workerCount = 5

func main() {
	env.Init()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	amqpURL := env.Get("AMQP_URL", "amqp://guest:guest@localhost/")
	rabbit, err := messaging.NewRabbitMQ(ctx, amqpURL)

	if err != nil {
		log.Fatal(err)
	}

	config := config.GetConfig()

	broker, err := rabbit.NewBroker(config.ExchangeName.Frontier)

	listerConfig := messaging.NewRabbitListenerConfig(
		config.ExchangeName.Frontier,
		fmt.Sprintf("%s.#", config.ExchangeName.Frontier),
	)

	listerConfig.PrefetchCount = workerCount
	listerConfig.Binding = false

	wk := worker.NewManager(ctx, workerCount, 5)
	wk.SpawnWorker()

	migrator := migrator{
		config: config,
		broker: broker,
	}

	for i := range workerCount {
		func(ctx context.Context, config messaging.RabbitListenerConfig) {
			wk.AssignTo(i, func() {
				listener, err := broker.NewListenerWithConfig(config)

				if err != nil {
					log.Println(err)
					return
				}

				log.Println("Start migration")
				if err := listener.Subscribe(ctx, migrator.handleMigration); err != nil {
					log.Println(err)
				}
			})
		}(ctx, listerConfig)
	}

	<-ctx.Done()
}
