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
	added_by INT NOT NULL REFERENCES users(user_id),
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