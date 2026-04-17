-- +goose Up
SELECT 'up SQL query';
ALTER TABLE domains
ADD CONSTRAINT scheme_host_port_unique_key UNIQUE (scheme, host, port);

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE domains
DROP CONSTRAINT scheme_host_port_unique_key;