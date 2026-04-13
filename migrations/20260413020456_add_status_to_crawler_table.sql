-- +goose Up
SELECT 'up SQL query';
CREATE TYPE crawler_status AS ENUM ('Alive', 'Unconscious', 'Dead');
ALTER TABLE crawlers 
ADD COLUMN status crawler_status NOT NULL DEFAULT 'Alive';

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE crawlers
DROP COLUMN status;
DROP TYPE crawler_status;
