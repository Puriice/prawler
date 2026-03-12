package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/puriice/golibs/pkg/pgutils"
	"github.com/purrice/prawler/internal/model"
)

var ()

type PostgresRobotsRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRobotsRepository(db *pgxpool.Pool) PostgresRobotsRepository {
	return PostgresRobotsRepository{db: db}
}

func (r PostgresRobotsRepository) AddRobots(context context.Context, host string, raw string) error {
	cmdTag, err := r.db.Exec(context, "INSERT INTO robots (host, raw_text) VALUES ($1, $2);", host, raw)

	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return pgutils.ErrNoRowsAffected
	}

	return nil
}

func (r PostgresRobotsRepository) GetRobots(context context.Context, host string) (*model.Robots, error) {
	var raw string

	err := r.db.QueryRow(context, "SELECT raw_text FROM robots WHERE host = $1", host).Scan(&raw)

	if err != nil {
		return nil, err
	}

	return &model.Robots{
		Host: host,
		Raw:  &raw,
	}, nil
}
