package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresBlacklistRepository struct {
	db *pgxpool.Pool
}

func NewPostgresBlacklistRepository(db *pgxpool.Pool) *PostgresBlacklistRepository {
	return &PostgresBlacklistRepository{
		db: db,
	}
}

func (r PostgresBlacklistRepository) Query(context context.Context) []string {
	rows, err := r.db.Query(context, "SELECT host FROM blacklists")

	if err != nil {
		return []string{}
	}

	blacklists, err := pgx.CollectRows(rows, pgx.RowTo[string])

	if err != nil {
		return []string{}
	}

	return blacklists
}
