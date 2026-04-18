package repository

import (
	"context"
	"net/url"
	"time"

	"github.com/purrice/prawler/internal/enum"
	"github.com/purrice/prawler/internal/html"
)

type CrawlerRepository interface {
	QueryCrawlerStatus(ctx context.Context) ([]CrawlerStatus, error)
	AddCrawler(context context.Context, uuid string, timestamp time.Time) error
	UpdateCrawlerStatus(ctx context.Context, uuid string, status string, lastSeen time.Time) error
	RemoveCrawler(ctx context.Context, uuid string) error
	AssignJob(ctx context.Context, crawlerUUID string, domain url.URL) error
}

type WebsiteRepository interface {
	GetBlacklistDomain(context context.Context) []string
	GetRobots(context context.Context, domain url.URL) (*string, *time.Time, error)
	GetParsedPage(context context.Context) []Page

	AddDomain(context context.Context, domain url.URL) error
	AddRobots(context context.Context, domain url.URL, raw string) error
	AddPage(context context.Context, url url.URL, depth int, page html.Page) (string, error)
	AddPageMetadata(context context.Context, pageUUID string, meta html.PageMetaData) error
	AddPageContent(context context.Context, pageUUID string, content html.PageContent) error

	SetPageStatus(context context.Context, pageUUID string, status enum.PageStatus) error
}
