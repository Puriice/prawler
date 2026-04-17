package repository

import (
	"context"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

func queryDomainUUID(context context.Context, db *pgxpool.Pool, domains url.URL) (string, error) {
	var uuid string

	err := db.QueryRow(context, "SELECT uuid FROM domains WHERE scheme = $1 AND host = $2 AND port = $3", domains.Scheme, domains.Hostname(), domains.Port()).Scan(&uuid)

	return uuid, err
}
