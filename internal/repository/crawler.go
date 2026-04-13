package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/puriice/golibs/pkg/pgutils"
)

type PostgresCrawlerRepository struct {
	db *pgxpool.Pool
}

func NewPostgresCrawlerRepository(db *pgxpool.Pool) *PostgresCrawlerRepository {
	return &PostgresCrawlerRepository{
		db: db,
	}
}

func (r *PostgresCrawlerRepository) UpdateCrawlerStatus(ctx context.Context, uuid string, status string, lastSeen time.Time) error {
	cmdTag, err := r.db.Exec(ctx, "INSERT INTO crawlers (uuid, status, last_seen) VALUES ($1, $2, $3) ON CONFLICT (uuid) DO UPDATE SET status = $2, last_seen = $3", uuid, status, lastSeen)

	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() != 1 {
		return pgutils.ErrNoRowsAffected
	}

	return nil
}

func (r *PostgresCrawlerRepository) RemoveCrawler(ctx context.Context, uuid string) error {
	cmdTag, err := r.db.Exec(ctx, "DELETE FROM crawlers WHERE uuid = $1", uuid)

	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() != 1 {
		return pgutils.ErrNoRowsAffected
	}

	return nil
}
