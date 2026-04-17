package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/puriice/golibs/pkg/pgutils"
)

type PostgresFrontierRepository struct {
	db *pgxpool.Pool
}

func NewPostgresFrontierRepository(db *pgxpool.Pool) PostgresFrontierRepository {
	return PostgresFrontierRepository{db: db}
}

func (r PostgresFrontierRepository) AddCrawler(context context.Context, uuid string, timestamp time.Time) error {
	cmdTag, err := r.db.Exec(context, "INSERT INTO crawlers (uuid, last_beats, created_at) VALUES ($1, $2, $2);", uuid, timestamp)

	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() != 1 {
		return pgutils.ErrNoRowsAffected
	}

	return nil
}
