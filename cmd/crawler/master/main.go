package main

import (
	"log"
	"time"

	"github.com/puriice/golibs/pkg/env"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/master"
)

func main() {
	env.Init()

	holter := heartbeat.NewHolter(5*time.Second, 5*time.Second, 2*time.Second)

	holter.Run()
	master.Run()

	log.Println("Shuting down")
}
