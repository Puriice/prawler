package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresWebRecordRepository struct {
	db *pgxpool.Pool
}

func NewPostgresWebRecordRepository(db *pgxpool.Pool) PostgresWebRecordRepository {
	return PostgresWebRecordRepository{db: db}
}

func (r PostgresWebRecordRepository) AddWebsite(context context.Context) error {
	return nil
}
