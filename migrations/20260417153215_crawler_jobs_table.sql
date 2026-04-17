-- +goose Up
SELECT 'up SQL query';
CREATE TABLE crawler_jobs (
	uuid			UUID DEFAULT uuidv7(),
	crawler_uuid	UUID NOT NULL,
	domain_uuid		UUID NOT NULL,
	created_at		TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	PRIMARY KEY(uuid),
	FOREIGN KEY(crawler_uuid) REFERENCES crawlers(uuid) ON UPDATE CASCADE ON DELETE CASCADE,
	FOREIGN KEY(domain_uuid) REFERENCES domains(uuid) ON UPDATE CASCADE ON DELETE CASCADE 
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE crawler_jobs;
