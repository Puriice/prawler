-- +goose Up
SELECT 'up SQL query';
ALTER TABLE pages 
ADD COLUMN http_status TEXT NOT NULL DEFAULT '200';

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE pages
DROP COLUMN http_status;
