-- +goose Up
SELECT 'up SQL query';
CREATE TYPE crawler_status AS ENUM('Alive', 'Unconscious', 'Dead');
CREATE TABLE crawlers (
	uuid 		UUID DEFAULT uuidv7(),
	status 		crawler_status DEFAULT 'Alive',
	last_seen 	TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_at 	TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	PRIMARY KEY(uuid)
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE crawlers;
DROP TYPE crawler_status;