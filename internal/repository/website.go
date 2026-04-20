package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/puriice/golibs/pkg/pgutils"
	"github.com/purrice/prawler/internal/enum"
	"github.com/purrice/prawler/internal/html"
	"github.com/purrice/prawler/internal/uri"
)

type PostgresWebsiteRepository struct {
	db *pgxpool.Pool
}

func NewPostgresWebsiteRepository(db *pgxpool.Pool) PostgresWebsiteRepository {
	return PostgresWebsiteRepository{db: db}
}

func (r PostgresWebsiteRepository) GetUnembeddedPages(context context.Context) ([]string, error) {
	rows, err := r.db.Query(
		context,
		`
		SELECT DISTINCT p.id::text
		FROM   pages p
		JOIN   chunks c ON c.page_id = p.id
		WHERE  p.noindex   = false
		AND  p.status   != 'failed'
		AND  p.status   != 'skipped'
		AND  c.embedding IS NULL
		`,
	)

	if err != nil {
		return []string{}, err
	}

	pageUUIDs, err := pgx.CollectRows(rows, pgx.RowTo[string])

	return pageUUIDs, err
}

func (r PostgresWebsiteRepository) GetBlacklistDomain(context context.Context) []string {
	rows, err := r.db.Query(context, "SELECT scheme, host, port FROM domains WHERE crawl_allowed = false")

	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var blacklists []string

	for rows.Next() {
		var scheme, host, port string

		if err := rows.Scan(&scheme, &host, &port); err != nil {
			continue
		}

		url, err := url.Parse(fmt.Sprintf("%s://%s:%s", scheme, host, port))

		if err != nil {
			continue
		}

		sitekey := uri.SiteKey(*url)

		blacklists = append(blacklists, sitekey)
	}

	if rows.Err() != nil {
		return []string{}
	}

	return blacklists
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

func (r PostgresWebsiteRepository) GetFinishedPage(context context.Context) []Page {
	rows, err := r.db.Query(
		context,
		`
		SELECT 
			url,
			canonical_url,
			indexable,
			depth,
			checksum
		FROM pages
		WHERE status = 'Parsed' OR status = 'Indexed' OR status = 'Skipped'
		`,
	)

	if err != nil {
		log.Println(err)
		return []Page{}
	}
	defer rows.Close()

	var pages []Page

	for rows.Next() {
		var url string
		var canonicalURL, checksum pgtype.Text
		var depth int
		var indexable bool

		rows.Scan(&url, &canonicalURL, &indexable, &depth, &checksum)

		pages = append(pages, Page{
			URL:   url,
			Depth: depth,
			Page: html.Page{
				CanonicalURL: canonicalURL.String,
				NoIndex:      !indexable,
				NoFollow:     false,
				Checksum:     checksum.String,
			},
		})
	}

	if err := rows.Err(); err != nil {
		log.Println(err)
		return []Page{}
	}

	return pages
}

func (r PostgresWebsiteRepository) GetPageContent(context context.Context, pageUUID string) (html.PageContent, error) {
	var content html.PageContent

	err := r.db.QueryRow(
		context,
		`
		SELECT raw_html, extracted_text, word_count 
		FROM page_content
		WHERE page_uuid = $1;
		`,
		pageUUID,
	).Scan(
		&content.RawHTML,
		&content.ExtractedText,
		&content.WordCount,
	)

	return content, err
}

func (r PostgresWebsiteRepository) AddDomain(context context.Context, domain url.URL) (string, error) {
	var uuid string

	err := r.db.QueryRow(
		context,
		`INSERT INTO domains (scheme, host, port) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (scheme, host, port) 
		DO UPDATE SET
			uuid = domains.uuid
		RETURNING uuid;
		`,
		domain.Scheme,
		domain.Hostname(),
		domain.Port(),
	).Scan(&uuid)

	if err != nil {
		return "", err
	}

	return uuid, nil
}

func (r PostgresWebsiteRepository) AddRobots(context context.Context, domains url.URL, raw string) error {
	var uuid string

	uuid, err := queryDomainUUID(context, r.db, domains)

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

func (r PostgresWebsiteRepository) AddPage(context context.Context, domainUUID string, url url.URL, depth int) (string, error) {
	var uuid string

	err := r.db.QueryRow(
		context,
		`
		INSERT INTO pages (domain_uuid, url, depth) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (domain_uuid, url)
		DO UPDATE SET
			domain_uuid = pages.domain_uuid
		RETURNING uuid
		`,
		domainUUID,
		url.String(),
		depth,
	).Scan(&uuid)

	return uuid, err
}

func (r PostgresWebsiteRepository) AddPageInformation(
	context context.Context,
	pageUUID string,
	url url.URL,
	depth int,
	page html.Page,
) error {
	_, err := r.db.Exec(
		context,
		`
		UPDATE pages
		SET
			url = $2, 
			canonical_url = $3,
			depth = $4,
			indexable = $5,
			checksum = $6
		WHERE 
			uuid = $1
		`,
		pageUUID,
		url.String(),
		page.CanonicalURL,
		depth,
		!page.NoIndex,
		page.Checksum,
	)

	return err
}

func (r PostgresWebsiteRepository) AddPageMetadata(context context.Context, pageUUID string, meta html.PageMetaData) error {
	var schemaOrgJSON []byte
	var schemaObjs []json.RawMessage

	if len(meta.SchemaOrg) > 0 {
		for _, blob := range meta.SchemaOrg {
			var raw json.RawMessage

			if err := json.Unmarshal([]byte(blob), &raw); err != nil {
				continue
			}
			schemaObjs = append(schemaObjs, raw)
		}

	}

	schemaOrgJSON, err := json.Marshal(schemaObjs)

	if err != nil {
		log.Printf("[WARN] Error marshal schema org json: %v", err)
	}

	_, err = r.db.Exec(
		context,
		`
		INSERT INTO page_metadata (page_uuid, title, language, description, author, published_at, schema_org) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) 
		ON CONFLICT (page_uuid) 
		DO UPDATE SET 
			title = $2,
			language = $3,
			description = $4,
			author = $5,
			published_at = $6,
			schema_org = $7;
		`,
		pageUUID,
		meta.Title,
		meta.Language,
		meta.MetaDescription,
		meta.Author,
		meta.PublishedAt,
		schemaOrgJSON,
	)

	if err != nil {
		return err
	}

	return nil
}

func (r PostgresWebsiteRepository) AddPageContent(context context.Context, pageUUID string, content html.PageContent) error {
	_, err := r.db.Exec(
		context,
		"INSERT INTO page_content (page_uuid, raw_html, extracted_text, word_count) VALUES ($1, $2, $3, $4)",
		pageUUID,
		content.RawHTML,
		content.ExtractedText,
		content.WordCount,
	)

	if err != nil {
		return err
	}

	if len(content.Chunks) == 0 {
		return nil
	}

	rows := make([][]any, 0, len(content.Chunks))

	for _, chunk := range content.Chunks {
		rows = append(rows, []any{pageUUID, chunk.Index, chunk.SectionHeading, chunk.Content, chunk.TokenEstimate})
	}

	r.db.CopyFrom(
		context,
		pgx.Identifier([]string{"chunks"}),
		[]string{
			"page_uuid",
			"chunk_index",
			"section_heading",
			"content",
			"token_count",
		},
		pgx.CopyFromRows(rows),
	)

	return err
}

func (r PostgresWebsiteRepository) AddLink(context context.Context, sourceUUID string, targetUUID string, anchorText string) error {
	_, err := r.db.Exec(
		context,
		`
		INSERT INTO links (source_page_uuid, target_page_uuid, anchor_text) 
		VALUES ($1, $2, $3)
		ON CONFLICT (source_page_uuid, target_page_uuid, anchor_text)
		DO NOTHING;
		`,
		sourceUUID,
		targetUUID,
		anchorText,
	)

	return err
}

func (r PostgresWebsiteRepository) SetPageStatus(context context.Context, pageUUID string, status enum.PageStatus) error {
	_, err := r.db.Exec(context, "UPDATE pages SET status = $2 WHERE uuid = $1", pageUUID, status)

	return err
}

func (r PostgresWebsiteRepository) BlacklistDomain(context context.Context, domain url.URL) error {
	_, err := r.db.Exec(
		context,
		`
			INSERT INTO domains (scheme, host, port, crawl_allowed) 
			VALUES ($1, $2, $3, FALSE)
			ON CONFLICT (scheme, host, port)
			DO UPDATE SET
				crawl_allowed = FALSE;
		`,
		domain.Scheme,
		domain.Hostname(),
		domain.Port(),
	)

	return err
}

func (r PostgresWebsiteRepository) UnBlacklistDomain(context context.Context, domain url.URL) error {
	_, err := r.db.Exec(
		context,
		`
			INSERT INTO domains (scheme, host, port, crawl_allowed) 
			VALUES ($1, $2, $3, TRUE)
			ON CONFLICT (scheme, host, port)
			DO UPDATE SET
				crawl_allowed = TRUE;
		`,
		domain.Scheme,
		domain.Hostname(),
		domain.Port(),
	)

	return err
}
