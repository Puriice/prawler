-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
CREATE TABLE crawlers (
	uuid 		UUID UNIQUE NOT NULL, 
	last_beats 	TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(uuid)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TABLE crawlers;
-- +goose StatementEnd
