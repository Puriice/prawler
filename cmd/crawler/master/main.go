package main

import (
	"log"
	"net/http"
	"time"

	"github.com/puriice/golibs/pkg/env"
	"github.com/puriice/golibs/pkg/server"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/master"
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
	master.Run()

	log.Println("Shuting down")
}
