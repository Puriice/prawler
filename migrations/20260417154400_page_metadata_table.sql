-- +goose Up
SELECT 'up SQL query';
CREATE TABLE page_metadata(
	page_uuid		UUID,
	title			TEXT,
	language		TEXT,
	description		TEXT,
	author			TEXT,
	published_at	TIMESTAMP,
	schema_org		JSONB,

	PRIMARY KEY(page_uuid),
	FOREIGN KEY(page_uuid) REFERENCES pages(uuid) ON UPDATE CASCADE ON DELETE CASCADE
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE page_metadata;