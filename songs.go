package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// // Date is a struct that allows custom unmarshalling behavior for dates
// type Date struct {
// 	*time.Time
// }

// // UnmarshalJSON sets time to nil for empty string, otherwise passes through to json.Unmarshal
// func (d *Date) UnmarshalJSON(b []byte) error {
// 	if bytes.Equal(b, []byte(`""`)) {
// 		d.Time = nil
// 		return nil
// 	}

// 	return json.Unmarshal(b, &d.Time)
// }

// // MarshalJSON marshals the custom Date object for encoding
// func (d *Date) MarshalJSON() ([]byte, error) {
// 	fmt.Println("Something must be wrong!")
// 	if d.Time == nil {
// 		return []byte("null"), nil
// 	}

// 	return json.Marshal(d.Time)
// }

// // MarshalJSON marshals the custom Date object for encoding
// func (d *Date) String() string {
// 	fmt.Println("String method called")
// 	if d.Time == nil {
// 		return ""
// 	}

// 	return d.Time.String()
// }

// Song is a struct that models the structure of a song, both in the request body, and in the DB
type Song struct {
	SongID        int64      `json:"song_id" db:"song_id"`
	Name          string     `json:"name" db:"name"`
	Artist        string     `json:"artist" db:"artist"`
	DateAdded     *time.Time `json:"date_added" db:"date_added"`
	Location      string     `json:"location" db:"location"`
	LastPerformed *string    `json:"last_performed,omitempty" db:"last_performed"`
	Notes         string     `json:"notes" db:"notes"`
	AddedBy       string     `json:"added_by" db:"added_by"`
	CollectionID  int64      `json:"collection_id" db:"collection_id"`
}

// TaggedSong is a struct that models tagging a song
type TaggedSong struct {
	TagID  int64 `json:"tag_id" db:"tag_id"`
	SongID int64 `json:"song_id" db:"song_id"`
}

// SongsHandler handles GETting all songs and POSTing a new song
func SongsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatalf("Unable to get session: %v\n", err)
		return
	}

	// Get URL parameter
	var collectionID int
	collectionID, err = strconv.Atoi(mux.Vars(r)["collection_id"])
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse collection id."}`))
		log.Println(err)
		return
	}

	if r.Method == "GET" {
		// Retrieve songs in collection
		rows, err := db.Query("SELECT song_id, name FROM songs WHERE collection_id = $1", collectionID)
		if err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error retrieving songs from database."}`))
			log.Println(err)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		songs := make([]Song, 0)
		for rows.Next() {
			var song Song
			if err := rows.Scan(&song.SongID, &song.Name); err != nil {
				log.Fatal(err)
			}
			songs = append(songs, song)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error retrieving songs from database."}`))
			log.Fatal(err)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(songs)
		return

	} else if r.Method == "POST" {
		// Parse and decode the request body into a new `Song` instance
		song := &Song{}
		if err := json.NewDecoder(r.Body).Decode(song); err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Unable to decode request body in SongsHandler (POST): %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Unable to decode request body."}`))
			return
		}

		// Input validation
		if song.Name == "" {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Cannot add a song with a blank name."}`))
			log.Println("Songs POST - Cannot create a song with a blank name.")
			return
		}

		// Create collection in database
		if _, err = db.Exec("INSERT INTO songs(name, artist, location, last_performed, notes, added_by, collection_id) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			song.Name, song.Artist, song.Location, song.LastPerformed, song.Notes, session.Values["email"], collectionID); err != nil {
			if err.(*pq.Error).Code == "23505" {
				// Song already exists
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Song already exists."}`))
				return
			}
			log.Printf("Unable to insert song record in database: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Unable to add song."}`))
			return
		}

		// Respond with success
		w.WriteHeader(http.StatusCreated)
	}
}

// SongHandler handles creating, updating, and deleting a single song.
func SongHandler(w http.ResponseWriter, r *http.Request) {
	// session, err := store.Get(r, "session")
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	log.Fatal(err)
	// 	return
	// }

	var song Song
	var err error

	// Get collection ID from URL
	song.CollectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse URL."}`))
		log.Printf("Song handler - Unable to parse collection id from URL: %v\n", err)
		return
	}

	// Get song ID from URL
	song.SongID, err = strconv.ParseInt(mux.Vars(r)["song_id"], 10, 64)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse URL."}`))
		log.Printf("Song handler - Unable to parse song id from URL: %v\n", err)
		return
	}

	if r.Method == "GET" {
		// Find the song in the database
		if err = db.QueryRow("SELECT songs.name, artist, location, last_performed, date_added, users.name, notes FROM songs JOIN users ON added_by = email WHERE collection_id = $1 AND songs.song_id = $2", song.CollectionID, song.SongID).Scan(&song.Name, &song.Artist, &song.Location, &song.LastPerformed, &song.DateAdded, &song.AddedBy, &song.Notes); err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(DATABASE_ERROR_MESSAGE)
				log.Printf("Song GET - Unable to get song from database: %v\n", err)
			}
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(song)
		return
	} else if r.Method == "PUT" {
		// Save the URL collection ID so the user can't update another record
		var collectionID = song.CollectionID

		err := json.NewDecoder(r.Body).Decode(&song)
		if err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Song PUT - Unable to parse request body: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Unable to parse request."}`))
			return
		}

		// Update song in database
		if *song.LastPerformed == "" {
			song.LastPerformed = nil
		}
		if _, err = db.Exec("UPDATE songs SET artist = $1, location = $2, last_performed = $3, notes = $4 WHERE collection_id = $5 AND name = $6", song.Artist, song.Location, song.LastPerformed, song.Notes, collectionID, song.Name); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Song PUT - Unable to update song in database: %v\n", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	} else if r.Method == "DELETE" {
		// Start db transaction
		tx, err := db.Begin()
		if err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Song DELETE - Unable to start database transaction: %v\n", err)
		}

		// // Remove songs from collection
		// if _, err = tx.Exec("DELETE FROM songs WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	w.Header().Add("Content-Type", "application/json")
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	w.Write([]byte(DATABASE_ERROR_MESSAGE))
		// 	log.Printf("Collection DELETE - Unable to delete songs from collection: %v\n", err)
		// 	return
		// }

		// // Remove tags from collection
		// if _, err = tx.Exec("DELETE FROM tags WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	w.Header().Add("Content-Type", "application/json")
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	w.Write([]byte(DATABASE_ERROR_MESSAGE))
		// 	log.Printf("Collection DELETE - Unable to delete tags from collection: %v\n", err)
		// 	return
		// }

		// // Remove users from collection
		// if _, err = tx.Exec("DELETE FROM collection_members WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	w.Header().Add("Content-Type", "application/json")
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	w.Write([]byte(DATABASE_ERROR_MESSAGE))
		// 	log.Printf("Collection DELETE - Unable to delete collection members: %v\n", err)
		// 	return
		// }

		// Delete collection
		if _, err = tx.Exec("DELETE FROM songs WHERE collection_id = $1 AND name = $2", song.CollectionID, song.Name); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Song DELETE - Unable to delete collection from database: %v\n", err)
			return
		}

		// Save changes
		if err = tx.Commit(); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Song DELETE - Unable to commit database transaction: %v\n", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}

// SongTagsHandler handles getting all tags from a particular song.
func SongTagsHandler(w http.ResponseWriter, r *http.Request) {
	var tags []Tag
	var songID, collectionID int64
	var err error
	var rows *sql.Rows

	session, err := store.Get(r, "session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatalf("Unable to get session: %v\n", err)
		return
	}

	// Get collection ID from URL
	collectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse URL."}`))
		log.Printf("Song handler - Unable to parse collection id from URL: %v\n", err)
		return
	}

	// Get song ID from URL
	songID, err = strconv.ParseInt(mux.Vars(r)["song_id"], 10, 64)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse URL."}`))
		log.Printf("Song handler - Unable to parse song id from URL: %v\n", err)
		return
	}

	if r.Method == "GET" {
		// Find the song in the database
		if rows, err = db.Query("SELECT tags.tag_id, name, description FROM tags JOIN tagged_songs ON tags.tag_id = tagged_songs.tag_id WHERE collection_id = $1 AND song_id = $2", collectionID, songID); err != nil {
			if err != sql.ErrNoRows {
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(DATABASE_ERROR_MESSAGE)
				log.Printf("Tagged song GET - Unable to get tagged songs from database: %v\n", err)
			}
			return
		}

		for rows.Next() {
			var tag Tag
			if err = rows.Scan(&tag.TagID, &tag.Name, &tag.Description); err != nil {
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(DATABASE_ERROR_MESSAGE)
				log.Printf("Tagged song GET - Unable to parse tagged song from database: %v\n", err)
			}

			tags = append(tags, tag)
		}

		if err = rows.Err(); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(DATABASE_ERROR_MESSAGE)
			log.Printf("Tagged song GET - Unable to get next tagged song from database: %v\n", err)
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(tags)
		return
	} else if r.Method == "POST" {
		var taggedSong TaggedSong

		err := json.NewDecoder(r.Body).Decode(&taggedSong)
		if err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("TaggedSong POST - Unable to parse request body: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Unable to parse request."}`))
			return
		}

		// Song ID Validation
		var targetCollectionID int64
		if err = db.QueryRow("SELECT collection_id FROM songs WHERE song_id = $1", songID).Scan(&targetCollectionID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Tag not found."}`))
		}

		if targetCollectionID != collectionID {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Song not found."}`))
			log.Printf("Tagged song POST - User %s (%s) attempted to tag song %d that they didn't own! Error %v\n", session.Values["name"], session.Values["email"], songID, err)
			return
		}

		// Tag ID Validation
		if err = db.QueryRow("SELECT collection_id FROM tags WHERE tag_id = $1", taggedSong.TagID).Scan(&targetCollectionID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Tag not found."}`))
		}

		if targetCollectionID != collectionID {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Tag not found."}`))
			log.Printf("Tagged song POST - User %s (%s) attempted to use tag %d that they didn't own! Error %v\n", session.Values["name"], session.Values["email"], taggedSong.TagID, err)
			return
		}

		// Create song tag in database
		log.Printf("Tag ID: %v\n", taggedSong.TagID)
		if _, err = db.Exec("INSERT INTO tagged_songs(tag_id, song_id) VALUES ($1, $2)",
			taggedSong.TagID, taggedSong.SongID); err != nil {
			if err.(*pq.Error).Code == "23505" {
				// Song is already tagged with this tag
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Song already tagged with this tag."}`))
				return
			}
			log.Printf("Tagged song POST - Unable to add tag to song record in database: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Unable to tag song."}`))
			return
		}

		// All operations completed successfully
		w.WriteHeader(http.StatusCreated)
		return
	} else if r.Method == "DELETE" {
		var taggedSong TaggedSong

		err := json.NewDecoder(r.Body).Decode(&taggedSong)
		if err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("TaggedSong POST - Unable to parse request body: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Unable to parse request."}`))
			return
		}

		// Song ID Validation
		var targetCollectionID int64
		if err = db.QueryRow("SELECT collection_id FROM songs WHERE song_id = $1", songID).Scan(&targetCollectionID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Tag not found."}`))
		}

		if targetCollectionID != collectionID {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Song not found."}`))
			log.Printf("Tagged song POST - User %s (%s) attempted to untag song %d that they didn't own! Error %v\n", session.Values["name"], session.Values["email"], songID, err)
			return
		}

		// Tag ID Validation
		if err = db.QueryRow("SELECT collection_id FROM tags WHERE tag_id = $1", taggedSong.TagID).Scan(&targetCollectionID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Tag not found."}`))
		}

		if targetCollectionID != collectionID {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Tag not found."}`))
			log.Printf("Tagged song POST - User %s (%s) attempted to untag %d that they didn't own! Error %v\n", session.Values["name"], session.Values["email"], taggedSong.TagID, err)
			return
		}

		// Delete song tag from database
		if _, err = db.Exec("DELETE FROM tagged_songs WHERE tag_id = $1 AND song_id = $2",
			taggedSong.TagID, songID); err != nil {
			log.Printf("Unable to delete song tag record from database: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Unable to remove tag from song."}`))
			return
		}

		// All operations completed successfully
		w.WriteHeader(http.StatusOK)
		return
	}
}
