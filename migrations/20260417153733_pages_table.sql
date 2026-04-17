-- +goose Up
SELECT 'up SQL query';
CREATE TYPE page_status AS ENUM('Pending', 'Parsed', 'Indexed', 'Failed', 'Skipped');
CREATE TABLE pages (
	uuid			UUID DEFAULT uuidv7(),
	domain_uuid		UUID NOT NULL,
	status			page_status NOT NULL DEFAULT 'Pending',
	depth			INT NOT NULL DEFAULT 0 CHECK (depth >= 0),
	url				TEXT NOT NULL,
	canonical_url	TEXT,
	check_sum		TEXT,
	updated_at		TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_at		TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	PRIMARY KEY(uuid),
	FOREIGN KEY(domain_uuid) REFERENCES domains(uuid)
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE pages;
DROP TYPE page_status;
