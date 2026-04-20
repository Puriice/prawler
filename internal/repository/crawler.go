package repository

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *PostgresCrawlerRepository) QueryCrawlerStatus(ctx context.Context) ([]CrawlerStatus, error) {
	rows, err := r.db.Query(ctx, "SELECT uuid, status, last_seen FROM crawlers")

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	crawlers, err := pgx.CollectRows(rows, pgx.RowToStructByName[CrawlerStatus])

	return crawlers, nil
}

func (r PostgresCrawlerRepository) AcquireDeadUUID(ctx context.Context) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string

	// Step 1: Try to grab a dead crawler (FOR UPDATE SKIP LOCKED avoids race conditions)
	query := `
		SELECT uuid
		FROM crawlers
		WHERE status = 'Dead'
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`

	err = tx.QueryRow(ctx, query).Scan(&id)
	if err == nil {
		// Step 2: Mark it as alive
		_, err = tx.Exec(
			ctx,
			`
			UPDATE crawlers
			SET 
				status = 'Alive',
				last_seen = $2
			WHERE uuid = $1
			`,
			id,
			time.Now(),
		)
		if err != nil {
			return "", err
		}

		if err := tx.Commit(ctx); err != nil {
			return "", err
		}

		return id, nil
	}

	// Step 3: If no dead crawler found → create new one
	if !errors.Is(err, pgx.ErrNoRows) && err.Error() != "no rows in result set" {
		return "", err
	}

	newID, err := uuid.NewV7()
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO crawlers (uuid, status, last_seen)
		VALUES ($1, 'Alive', $2)
		`,
		newID,
		time.Now(),
	)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return newID.String(), nil
}

func (r PostgresCrawlerRepository) AddCrawler(context context.Context, uuid string, timestamp time.Time) error {
	cmdTag, err := r.db.Exec(context, "INSERT INTO crawlers (uuid, last_beats, created_at) VALUES ($1, $2, $2);", uuid, timestamp)

	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() != 1 {
		return pgutils.ErrNoRowsAffected
	}

	return nil
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

func (r *PostgresCrawlerRepository) AssignJob(ctx context.Context, crawlerUUID string, domain url.URL) error {
	domainUUID, err := queryDomainUUID(ctx, r.db, domain)

	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, "INSERT INTO crawler_jobs (crawler_uuid, domain_uuid) VALUES ($1, $2) ON CONFLICT(domain_uuid) DO UPDATE SET crawler_uuid = $1;", crawlerUUID, domainUUID)

	if err != nil {
		return err
	}

	return nil
}
