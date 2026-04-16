package repository

import "time"

type CrawlerStatus struct {
	UUID     string    `db:"uuid"`
	Status   string    `db:"status"`
	LastSeen time.Time `db:"last_seen"`
}
