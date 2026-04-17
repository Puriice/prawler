-- +goose Up
SELECT 'up SQL query';
ALTER TABLE pages 
RENAME COLUMN check_sum TO checksum;

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE pages
RENAME COLUMN checksum TO check_sum;