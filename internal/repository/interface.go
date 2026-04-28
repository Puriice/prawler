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
	AcquireDeadUUID(context context.Context) (string, error)
	UpdateCrawlerStatus(ctx context.Context, uuid string, status string, lastSeen time.Time) error
	RemoveCrawler(ctx context.Context, uuid string) error
	AssignJob(ctx context.Context, crawlerUUID string, domain url.URL) error
}

type WebsiteRepository interface {
	GetBlacklistDomain(context context.Context) []string
	GetRobots(context context.Context, domain url.URL) (*string, *time.Time, error)
	GetFinishedPage(context context.Context) []Page
	GetLinks(context context.Context, pageUUID string) []string
	GetPageCache(context context.Context, pageUUID string) (int, *html.PageContent, error)
	GetPageContent(context context.Context, pageUUID string) (html.PageContent, error)

	AddDomain(context context.Context, domain url.URL) (string, error)
	AddRobots(context context.Context, domain url.URL, raw string) error
	AddPage(context context.Context, domainUUID string, url url.URL, depth int) (string, error)
	AddPageInformation(context context.Context, pageUUID string, url url.URL, depth int, page html.Page) error
	AddPageMetadata(context context.Context, pageUUID string, meta html.PageMetaData) error
	AddPageContent(context context.Context, pageUUID string, content html.PageContent) error
	AddLink(context context.Context, sourceUUID string, targetUUID string, anchorText string) error

	SetPageStatus(context context.Context, pageUUID string, status enum.PageStatus) error
	BlacklistDomain(context context.Context, domain url.URL) error
	UnBlacklistDomain(context context.Context, domain url.URL) error
}
