-- +goose Up
SELECT 'up SQL query';
CREATE TABLE robots (
	host 		TEXT NOT NULL,
	raw_text	TEXT NOT NULL,
	
	PRIMARY KEY(host)
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE robots;