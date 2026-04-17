package repository

import (
	"context"
	"net/url"
	"time"
)

type CrawlerRepository interface {
	QueryCrawlerStatus(ctx context.Context) ([]CrawlerStatus, error)
	AddCrawler(context context.Context, uuid string, timestamp time.Time) error
	UpdateCrawlerStatus(ctx context.Context, uuid string, status string, lastSeen time.Time) error
	RemoveCrawler(ctx context.Context, uuid string) error
	AssignJob(ctx context.Context, crawlerUUID string, domain url.URL) error
}

type WebsiteRepository interface {
	GetRobots(context context.Context, domain url.URL) (*string, *time.Time, error)

	AddDomain(context context.Context, domain url.URL) error
	AddRobots(context context.Context, domain url.URL, raw string) error
	AddPage(context context.Context, domain url.URL, url string, canonical_url string, depth int, indexable bool, checksum string) (string, error)
}

type BlacklistRepository interface {
	Query(context context.Context) []string
}
