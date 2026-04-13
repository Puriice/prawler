package master

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
)

func Run(ctx context.Context, db *pgxpool.Pool, mux *http.ServeMux, listener messaging.RabbitListener) {
	fetcher := fetch.NewFecter(nil)

	robotsRepository := repository.NewPostgresRobotsRepository(db)
	robotParser := robots.NewRobotParser(robotsRepository, &fetcher)

	crawlerRepository := repository.NewPostgresCrawlerRepository(db)
	masterHandler := NewHTTPHandler(ctx, crawlerRepository)
	masterHandler.RegisterHandler(mux)
	master := NewMasterNode(ctx, db, &robotParser)

	log.Println("Start listening to slave producing events.")
	if err := listener.Subscribe(ctx, master.Handle); err != nil {
		log.Println(err)
	}
}
