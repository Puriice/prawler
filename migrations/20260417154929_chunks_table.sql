-- +goose Up
SELECT 'up SQL query';
CREATE EXTENSION vector;
CREATE TABLE chunks (
	uuid			UUID DEFAULT uuidv7(),
	page_uuid		UUID,
	chunk_index		INT,
	section_heading TEXT,
	content			TEXT,
	token_count		INT,
	embedding		VECTOR(1536),

	PRIMARY KEY(uuid),
	FOREIGN KEY(page_uuid) REFERENCES pages(uuid) ON UPDATE CASCADE ON DELETE CASCADE
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE chunks;