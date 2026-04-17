-- +goose Up
SELECT 'up SQL query';
ALTER TABLE pages
ADD COLUMN indexable BOOLEAN NOT NULL;

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE pages
DROP COLUMN indexable;
