-- +goose Up
SELECT 'up SQL query';
CREATE TYPE crawling_status AS ENUM ('uncrawl', 'crawling', 'crawled', 'blacklisted');
CREATE TABLE sites (
	host TEXT,
	last_crawler UUID,
	status crawling_status,
	updated_at TIMESTAMP,
	created_at TIMESTAMP, 
	PRIMARY KEY(host),
	FOREIGN KEY(last_crawler) REFERENCES crawlers(uuid)
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE sites;
DROP TYPE crawling_status;