CREATE TABLE IF NOT EXISTS users
(
	user_id SERIAL PRIMARY KEY,
	email VARCHAR(255) UNIQUE NOT NULL,
	password VARCHAR(127) NOT NULL,
	name VARCHAR(127) NOT NULL,
	verified BOOLEAN DEFAULT false,
	restricted BOOLEAN DEFAULT false,
	created TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	last_login TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS verification_emails
(
	user_id INT PRIMARY KEY REFERENCES users(user_id),
	token CHAR(64) NOT NULL,
	expires TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP + interval '24 hours'
);

CREATE TABLE IF NOT EXISTS collections
(
	collection_id SERIAL PRIMARY KEY,
	name VARCHAR(127) NOT NULL,
	description VARCHAR(1023)
);

CREATE TABLE IF NOT EXISTS collection_members
(
	user_id INT REFERENCES users(user_id),
	collection_id INT REFERENCES collections(collection_id),
	admin BOOLEAN,
	PRIMARY KEY (user_id, collection_id)
);

CREATE TABLE IF NOT EXISTS invitations
(
	invitation_id SERIAL PRIMARY KEY,
	inviter_id INT REFERENCES users(user_id) NOT NULL,
	invitee_email VARCHAR(255) NOT NULL,
	collection_id INT REFERENCES collections(collection_id) NOT NULL,
	admin_invite BOOLEAN NOT NULL,
	invite_sent TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
	retracted BOOLEAN NOT NULL DEFAULT FALSE,
	token CHAR(64) NOT NULL,
	UNIQUE (invitee_email, collection_id, inviter_id)
);

CREATE TABLE IF NOT EXISTS password_reset
(
	user_id INT PRIMARY KEY REFERENCES users(user_id),
	token CHAR(64) NOT NULL,
	expires TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS songs
(
	song_id SERIAL PRIMARY KEY,
	name VARCHAR(127) NOT NULL,
	artist VARCHAR(127),
	date_added DATE NOT NULL DEFAULT CURRENT_DATE,
	location VARCHAR(127),
	last_performed DATE,
	notes TEXT,
	added_by INT REFERENCES users(user_id),
	collection_id INT NOT NULL REFERENCES collections(collection_id)
);

CREATE TABLE IF NOT EXISTS tags
(
	tag_id SERIAL PRIMARY KEY,
	name VARCHAR(127) NOT NULL,
	description TEXT,
	collection_id INT NOT NULL REFERENCES collections(collection_id)
);

CREATE TABLE IF NOT EXISTS tagged_songs
(
	song_id INT REFERENCES songs(song_id),
	tag_id INT REFERENCES tags(tag_id),
	PRIMARY KEY (song_id, tag_id)
);

CREATE OR REPLACE FUNCTION search_collection(collection_id INT, query VARCHAR(255))
RETURNS TABLE (song_id INT, song_name VARCHAR(127))
LANGUAGE plpgsql
AS $$
DECLARE
-- variable declaration
BEGIN
RETURN QUERY
	SELECT s_search.song_id, s_search.song_name
	FROM (SELECT s.song_id as song_id,
			     s.name as song_name,
			     setweight(to_tsvector(s.name), 'A') ||
			     setweight(to_tsvector(s.artist), 'B') ||
			     setweight(to_tsvector(s.location), 'B') ||
			     setweight(to_tsvector(s.notes), 'B') ||
			     setweight(to_tsvector(coalesce(string_agg(t.name, ' '), '')), 'C') AS document
		FROM songs AS s
		JOIN tagged_songs AS ts ON s.song_id = ts.song_id
		JOIN tags AS t ON t.tag_id = ts.tag_id
		WHERE s.collection_id = search_collection.collection_id
		GROUP BY s.song_id) s_search
	WHERE s_search.document @@ to_tsquery(query)
    ORDER BY ts_rank(s_search.document, to_tsquery(query)) DESC;
END; $$