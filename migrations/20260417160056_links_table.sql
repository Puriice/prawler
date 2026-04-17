-- +goose Up
SELECT 'up SQL query';
CREATE TABLE links (
	uuid				UUID DEFAULT uuidv7(),
	source_page_uuid	UUID NOT NULL,
	target_page_uuid	UUID NOT NULL,
	anchor_text			TEXT,
	created_at			TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	PRIMARY KEY(uuid),
	FOREIGN KEY(source_page_uuid) REFERENCES pages(uuid) ON UPDATE CASCADE ON DELETE CASCADE,
	FOREIGN KEY(target_page_uuid) REFERENCES pages(uuid) ON UPDATE CASCADE ON DELETE CASCADE
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE links;