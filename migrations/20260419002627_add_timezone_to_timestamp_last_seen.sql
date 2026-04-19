-- +goose Up
SELECT 'up SQL query';
ALTER TABLE crawlers
ALTER COLUMN last_seen TYPE TIMESTAMP WITH TIME ZONE;

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE crawlers
ALTER COLUMN last_seen TYPE TIMESTAMP WITHOUT TIME ZONE;
