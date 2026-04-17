-- +goose Up
SELECT 'up SQL query';
CREATE TABLE page_content (
	page_uuid		UUID,
	raw_html		TEXT,
	extracted_text	TEXT,
	word_count		INT,
	created_at		TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	PRIMARY KEY(page_uuid),
	FOREIGN KEY(page_uuid) REFERENCES pages(uuid) ON UPDATE CASCADE ON DELETE CASCADE
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE page_content;