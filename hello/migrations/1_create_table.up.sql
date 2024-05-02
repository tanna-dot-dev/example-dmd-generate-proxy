CREATE TABLE packages_ecosystems_responses(
	url TEXT NOT NULL PRIMARY KEY,
	data JSON NOT NULL,
	last_updated TEXT NOT NULL
);
