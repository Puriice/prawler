package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/puriice/golibs/pkg/pgutils"
)

type PostgresfrontierRepository struct {
	db *pgxpool.Pool
}

func NewPostgresfrontierRepository(db *pgxpool.Pool) PostgresfrontierRepository {
	return PostgresfrontierRepository{db: db}
}

func (r PostgresfrontierRepository) AddCrawler(context context.Context, uuid string, timestamp time.Time) error {
	cmdTag, err := r.db.Exec(context, "INSERT INTO crawlers (uuid, last_beats, created_at) VALUES ($1, $2, $2);", uuid, timestamp)

	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() != 1 {
		return pgutils.ErrNoRowsAffected
	}

	return nil
}
