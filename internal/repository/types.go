package repository

import (
	"time"

	"github.com/purrice/prawler/internal/html"
)

type CrawlerStatus struct {
	UUID     string    `db:"uuid"`
	Status   string    `db:"status"`
	LastSeen time.Time `db:"last_seen"`
}

type Page struct {
	html.Page
	URL   string
	Depth int
}
