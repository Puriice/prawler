-- +goose Up
SELECT 'up SQL query';
ALTER TABLE pages
ADD CONSTRAINT domain_url_unique_key UNIQUE(domain_uuid, url);

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE pages
DROP CONSTRAINT domain_url_unique_key;