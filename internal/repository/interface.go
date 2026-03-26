package repository

import (
	"context"
	"time"
)

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
