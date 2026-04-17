-- +goose Up
SELECT 'up SQL query';
CREATE TYPE crawling_status AS ENUM('Uncrawl', 'Crawling', 'Crawled');
CREATE TABLE domains (
	uuid			UUID DEFAULT uuidv7(),
	scheme			TEXT,
	host			TEXT,
	port			TEXT,
	status			crawling_status DEFAULT 'Uncrawl',
	crawl_allowed	BOOLEAN,
	updated_at		TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_at		TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	PRIMARY KEY(uuid)
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE domains;
DROP TYPE crawling_status;