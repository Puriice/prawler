package repository

import (
	"context"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/puriice/golibs/pkg/pgutils"
)

type PostgresWebsiteRepository struct {
	db *pgxpool.Pool
}

func NewPostgresWebsiteRepository(db *pgxpool.Pool) PostgresWebsiteRepository {
	return PostgresWebsiteRepository{db: db}
}

func (r PostgresWebsiteRepository) AddDomain(context context.Context, domain url.URL) error {
	_, err := r.db.Exec(context, "INSERT INTO domains (scheme, host, port) VALUES ($1, $2, $3) ON CONFLICT (scheme, host, port) DO NOTHING", domain.Scheme, domain.Hostname(), domain.Port())

	if err != nil {
		return err
	}

	return nil
}

func (r PostgresWebsiteRepository) AddRobots(context context.Context, domains url.URL, raw string) error {
	var uuid string

	err := r.db.QueryRow(context, "SELECT uuid FROM domains WHERE scheme = $1 AND host = $2 AND port = $3", domains.Scheme, domains.Hostname(), domains.Port()).Scan(&uuid)

	if err != nil {
		return err
	}

	cmdTag, err := r.db.Exec(context, "INSERT INTO robots (uuid, raw_text) VALUES ($1, $2) ON CONFLICT(uuid) DO UPDATE SET raw_text = $2, updated_at = CURRENT_TIMESTAMP;", uuid, raw)

	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return pgutils.ErrNoRowsAffected
	}

	return nil
}

func (r PostgresWebsiteRepository) GetRobots(context context.Context, domain url.URL) (*string, *time.Time, error) {
	var raw string
	var timestamp time.Time

	err := r.db.QueryRow(context, "SELECT r.raw_text, r.updated_at FROM robots r LEFT JOIN domains d WHERE d.scheme = $1 AND d.host = $2 AND d.port = $3", domain.Scheme, domain.Hostname(), domain.Port()).Scan(&raw, &timestamp)

	if err != nil {
		return nil, nil, err
	}

	return &raw, &timestamp, nil
}
