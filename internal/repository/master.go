package repository

import "github.com/jackc/pgx/v5/pgxpool"

type PostgresMasterRepository struct {
	db *pgxpool.Pool
}

func NewPostgresMasterRepository(db *pgxpool.Pool) PostgresMasterRepository {
	return PostgresMasterRepository{db: db}
}
