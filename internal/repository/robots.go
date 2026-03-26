package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/puriice/golibs/pkg/pgutils"
)

var ()

type PostgresRobotsRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRobotsRepository(db *pgxpool.Pool) PostgresRobotsRepository {
	return PostgresRobotsRepository{db: db}
}

func (r PostgresRobotsRepository) AddRobots(context context.Context, host string, raw string) error {
	cmdTag, err := r.db.Exec(context, "INSERT INTO robots (host, raw_text) VALUES ($1, $2) ON DUPLICATE KEY UPDATE raw_text = $2, updated_at = CURRENT_TIMESTAMP;", host, raw)

	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return pgutils.ErrNoRowsAffected
	}

	return nil
}

func (r PostgresRobotsRepository) GetRobots(context context.Context, host string) (*string, *time.Time, error) {
	var raw string
	var timestamp time.Time

	err := r.db.QueryRow(context, "SELECT raw_text, updated_at FROM robots WHERE host = $1", host).Scan(&raw, &timestamp)

	if err != nil {
		return nil, nil, err
	}

	return &raw, &timestamp, nil
}
