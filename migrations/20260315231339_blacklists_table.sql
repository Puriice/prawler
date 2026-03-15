-- +goose Up
SELECT 'up SQL query';
CREATE TABLE blacklists (
	host		TEXT UNIQUE NOT NULL,
	created_at	TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	PRIMARY KEY (host)
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE blacklists;