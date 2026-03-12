package repository

import "context"

type WebRecordRepository interface {
	AddWebsite(context context.Context) error
}

type WebInfoRepository interface {
	AddInfo(context context.Context) error
}
