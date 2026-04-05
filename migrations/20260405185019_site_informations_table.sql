-- +goose Up
SELECT 'up SQL query';
CREATE EXTENSION vector;
CREATE TABLE site_informations(
	host TEXT,
	path TEXT,
	query TEXT,
	keywords TEXT[],
	description TEXT,
	raw_text TEXT,
	embedding VECTOR(1536),
	updated_at TIMESTAMP,
	created_at TIMESTAMP,
	PRIMARY KEY(host, path, query)
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE site_informations;
DROP EXTENSION vector;