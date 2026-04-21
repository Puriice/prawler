-- +goose Up
SELECT 'up SQL query';
ALTER TABLE pages
ADD COLUMN links TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE pages
DROP COLUMN links;