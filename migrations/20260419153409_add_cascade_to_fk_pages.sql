-- +goose Up
SELECT 'up SQL query';
ALTER TABLE pages
DROP CONSTRAINT pages_domain_uuid_fkey;
ALTER TABLE pages
ADD CONSTRAINT pages_domain_uuid_fkey
FOREIGN KEY (domain_uuid)
REFERENCES domains(uuid)
ON UPDATE CASCADE
ON DELETE CASCADE;

-- +goose Down
SELECT 'down SQL query';
ALTER TABLE pages
DROP CONSTRAINT pages_domain_uuid_fkey;
ALTER TABLE pages
ADD CONSTRAINT pages_domain_uuid_fkey
FOREIGN KEY (domain_uuid)
REFERENCES domains(uuid);