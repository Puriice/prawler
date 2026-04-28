package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/puriice/golibs/pkg/db"
	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/events"
	"github.com/purrice/prawler/internal/repository"
)

func main() {
	env.Init()
	db, err := db.NewDatabase()

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	amqpURL := env.Get("amqp_url", "amqp://guest:guest@localhost:5672")
	config := config.GetConfig()

	rabbit, err := messaging.NewRabbitMQ(context.Background(), amqpURL)

	if err != nil {
		db.Close()
		log.Fatal(err)
	}

	defer db.Close()

	embeddingBroker, err := rabbit.NewBroker(config.ExchangeName.Embedding)

	if err != nil {
		db.Close()
		rabbit.Shutdown()
		log.Fatal(err)
	}

	defer rabbit.Shutdown()

	website := repository.NewPostgresWebsiteRepository(db)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pageUUIDs, err := website.GetUnembeddedPages(ctx)

	if err != nil {
		stop()
		db.Close()
		rabbit.Shutdown()
		log.Fatal(err)
	}

	for _, uuid := range pageUUIDs {
		embeddingBroker.Publish(fmt.Sprintf("%s.uuid", config.ExchangeName.Embedding), events.EmbeddingEvent{
			Type:     events.EventEmbedding,
			PageUUID: uuid,
		})
	}

}
