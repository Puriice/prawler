-- +goose Up
SELECT 'up SQL query';
ALTER TABLE domains
ALTER COLUMN crawl_allowed SET DEFAULT TRUE;

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE domains
ALTER COLUMN crawl_allowed DROP DEFAULT;