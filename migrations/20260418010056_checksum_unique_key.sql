-- +goose Up
SELECT 'up SQL query';
ALTER TABLE pages 
ADD CONSTRAINT checksum_unique_key UNIQUE(url, checksum);

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE pages 
DROP CONSTRAINT checksum_unique_key;