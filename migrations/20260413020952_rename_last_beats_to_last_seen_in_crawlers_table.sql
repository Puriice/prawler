-- +goose Up
SELECT 'up SQL query';
ALTER TABLE crawlers
RENAME last_beats TO last_seen;

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE crawlers
RENAME last_seen TO last_beats;
