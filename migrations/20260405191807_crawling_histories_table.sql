-- +goose Up
SELECT 'up SQL query';
CREATE TABLE crawling_histories(
	id INT GENERATED ALWAYS AS IDENTITY,
	crawler UUID,
	host TEXT,
	path TEXT,
	query TEXT,
	crawl_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(id),
	FOREIGN KEY(host, path, query) REFERENCES site_informations(host, path, query)
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE crawling_histories;