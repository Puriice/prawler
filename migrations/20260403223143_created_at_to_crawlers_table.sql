-- +goose Up
SELECT 'up SQL query';
ALTER TABLE crawlers 
ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE crawlers
DROP COLUMN created_at;
