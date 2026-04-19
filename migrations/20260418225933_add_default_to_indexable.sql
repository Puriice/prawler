-- +goose Up
SELECT 'up SQL query';
ALTER TABLE pages
ALTER COLUMN indexable SET DEFAULT TRUE;

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE pages
ALTER COLUMN indexable DROP DEFAULT;
