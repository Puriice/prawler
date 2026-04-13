package master

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
)

func Run(db *pgxpool.Pool, listener messaging.RabbitListener) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fetcher := fetch.NewFecter(nil)

	robotsRepository := repository.NewPostgresRobotsRepository(db)
	robotParser := robots.NewRobotParser(robotsRepository, &fetcher)

	master := NewMasterNode(ctx, db, &robotParser)

	log.Println("Start listening to slave producing events.")
	if err := listener.Subscribe(ctx, master.Handle); err != nil {
		log.Println(err)
	}
}
