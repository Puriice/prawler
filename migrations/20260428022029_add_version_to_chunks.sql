-- +goose Up
SELECT 'up SQL query';
ALTER TABLE chunks
ADD COLUMN embedding_version INT NOT NULL DEFAULT 1;
 
-- +goose Down
SELECT 'down SQL query';

ALTER TABLE chunks
DROP COLUMN embedding_version;