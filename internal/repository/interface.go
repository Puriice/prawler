package repository

import (
	"context"

	"github.com/purrice/prawler/internal/model"
)

type RobotsRepository interface {
	AddRobots(context context.Context, host string, raw string) error
	GetRobots(context context.Context, host string) (*model.Robots, error)
}

type WebRecordRepository interface {
	AddWebsite(context context.Context) error
}

type WebInfoRepository interface {
	AddInfo(context context.Context) error
}
