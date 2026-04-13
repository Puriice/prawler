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
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/master"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
)

func handleHeartbeat() *heartbeat.Holter {
	holter := heartbeat.NewHolter(5*time.Second, 5*time.Second, 2*time.Second)
	go holter.Monitor()

	host := env.Get("HOST", "")
	port := env.Get("PORT", "5723")

	server := server.NewServer(host, port, nil)

	handler := http.NewServeMux()
	handler.HandleFunc("/heartbeat", holter.HandleBeats)
	handler.HandleFunc("/nodes", holter.HandleList)
	handler.HandleFunc("/", holter.HandleDashboard)

	server.Handler = handler

	go server.Start()

	return holter
}

func main() {
	env.Init()

	handleHeartbeat()

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
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fetcher := fetch.NewFecter(nil)

	robotsRepository := repository.NewPostgresRobotsRepository(db)
	robotParser := robots.NewRobotParser(robotsRepository, &fetcher)

	master := master.NewMasterNode(ctx, db, &robotParser)

	log.Println("Start listening to slave producing events.")
	if err := listener.Subscribe(ctx, master.Handle); err != nil {
		log.Println(err)
	}

	log.Println("Shuting down")
}
