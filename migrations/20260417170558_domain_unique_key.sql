-- +goose Up
SELECT 'up SQL query';
ALTER TABLE crawler_jobs
ADD CONSTRAINT domain_unique_key UNIQUE (domain_uuid);


-- +goose Down
SELECT 'down SQL query';
ALTER TABLE crawler_jobs
DROP CONSTRAINT domain_unique_key;