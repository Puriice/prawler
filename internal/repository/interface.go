package repository

import (
	"context"
	"time"
)

type CrawlerRepository interface {
	QueryCrawlerStatus(ctx context.Context) ([]CrawlerStatus, error)
	UpdateCrawlerStatus(ctx context.Context, uuid string, status string, lastSeen time.Time) error
	RemoveCrawler(ctx context.Context, uuid string) error
}

type RobotsRepository interface {
	AddRobots(context context.Context, host string, raw string) error
	GetRobots(context context.Context, host string) (*string, *time.Time, error)
}

type WebRecordRepository interface {
	AddWebsite(context context.Context) error
}

type WebInfoRepository interface {
	AddInfo(context context.Context) error
}

type BlacklistRepository interface {
	Query(context context.Context) []string
}

type MasterRepository interface {
	AddCrawler(context context.Context, uuid string, timestamp time.Time) error
}
