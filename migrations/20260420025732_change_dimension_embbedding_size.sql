-- +goose Up
SELECT 'up SQL query';
DROP INDEX IF EXISTS chunks_embedding_idx;
ALTER TABLE chunks ALTER COLUMN embedding TYPE vector(1024);
CREATE INDEX chunks_embedding_idx ON chunks
USING hnsw (embedding vector_cosine_ops);

-- +goose Down
SELECT 'down SQL query';
DROP INDEX IF EXISTS chunks_embedding_idx;
ALTER TABLE chunks ALTER COLUMN embedding TYPE vector(1536);
CREATE INDEX chunks_embedding_idx ON chunks
USING hnsw (embedding vector_cosine_ops);