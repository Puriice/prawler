-- +goose Up
SELECT 'up SQL query';
ALTER TABLE links 
ADD CONSTRAINT source_target_unique_key UNIQUE(source_page_uuid, target_page_uuid, anchor_text);

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE links 
DROP CONSTRAINT source_target_unique_key;