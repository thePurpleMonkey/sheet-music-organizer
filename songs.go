package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
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

	// Order of song in setlist
	Order int64 `json:"order,omitempty" db:"order"`
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
		log.Printf("Songs handler - Unable to get session: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Get URL parameter
	var collectionID int
	collectionID, err = strconv.Atoi(mux.Vars(r)["collection_id"])
	if err != nil {
		log.Printf("Songs handler - Unable to parse collection ID from URL: %v\n", err)
		SendError(w, `{"error": "Unable to parse URL."}`, http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		// Get excluded tags list
		var excludedTags []int64
		var queryString string = r.URL.Query().Get("exclude_tags")
		if len(queryString) > 0 {
			log.Printf("Songs GET - Query string: '%v'\n", queryString)
			if err := json.Unmarshal([]byte(queryString), &excludedTags); err != nil {
				log.Printf("Songs GET - Unable to get list of excluded tags from query string: %v\n", err)
				SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
				return
			}
			log.Printf("Songs GET - Excluded tags: %v\n", excludedTags)
		}

		// Retrieve songs in collection
		// rows, err := db.Query("SELECT song_id, name, date_added FROM songs WHERE collection_id = $1", collectionID)
		rows, err := db.Query(`
			SELECT s.song_id, s.name, s.date_added 
			FROM songs AS s 
			LEFT JOIN tagged_songs AS ts 
				ON s.song_id = ts.song_id 
			   AND ts.tag_id = ANY($2) 
			WHERE s.collection_id = $1 
			AND ts.tag_id IS NULL`, collectionID, pq.Array(excludedTags))
		if err != nil {
			log.Printf("Songs GET - Unable to get songs from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		songs := make([]Song, 0)
		for rows.Next() {
			var song Song
			if err := rows.Scan(&song.SongID, &song.Name, &song.DateAdded); err != nil {
				log.Printf("Songs GET - Unable to get song from database result: %v\n", err)
			}
			songs = append(songs, song)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Songs GET - Unable to get songs from database result: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
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
			log.Printf("Songs POST - Unable to decode request body: %v\n", err)
			body, _ := ioutil.ReadAll(r.Body)
			log.Printf("Body: %s\n", body)
			SendError(w, `{"error": "Unable to decode request body."}`, http.StatusBadRequest)
			return
		}

		// Input validation
		if song.Name == "" {
			log.Println("Songs POST - Cannot create a song with a blank name.")
			SendError(w, `{"error": "Cannot add a song with a blank name."}`, http.StatusBadRequest)
			return
		}

		// Create collection in database
		if _, err = db.Exec("INSERT INTO songs(name, artist, location, last_performed, notes, added_by, collection_id) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			song.Name, song.Artist, song.Location, song.LastPerformed, song.Notes, session.Values["user_id"], collectionID); err != nil {
			if err.(*pq.Error).Code == "23505" {
				// Song already exists
				w.Header().Add("Content-Type", "application/json")
				SendError(w, `{"error": "Song already exists."}`, http.StatusBadRequest)
				return
			}
			log.Printf("Songs POST - Unable to insert song record in database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Respond with success
		w.WriteHeader(http.StatusCreated)
	}
}

// SongHandler handles creating, updating, and deleting a single song.
func SongHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		log.Printf("Song handler - Unable to get session: %v\n", err)
		return
	}

	var song Song

	// Get collection ID from URL
	song.CollectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		log.Printf("Song handler - Unable to parse collection id from URL: %v\n", err)
		SendError(w, `{"error": "Unable to parse URL."}`, http.StatusBadRequest)
		return
	}

	// Get song ID from URL
	song.SongID, err = strconv.ParseInt(mux.Vars(r)["song_id"], 10, 64)
	if err != nil {
		log.Printf("Song handler - Unable to parse song id from URL: %v\n", err)
		SendError(w, `{"error": "Unable to parse URL."}`, http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		// Find the song in the database
		if err = db.QueryRow("SELECT songs.name, artist, location, last_performed, date_added, users.name, notes FROM songs JOIN users ON added_by = user_id WHERE collection_id = $1 AND songs.song_id = $2", song.CollectionID, song.SongID).Scan(&song.Name, &song.Artist, &song.Location, &song.LastPerformed, &song.DateAdded, &song.AddedBy, &song.Notes); err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
			} else {
				log.Printf("Song GET - Unable to get song from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
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
			body, _ := ioutil.ReadAll(r.Body)
			log.Printf("Body: %s\n", body)
			SendError(w, `{"error": "Unable to parse request."}`, http.StatusBadRequest)
			return
		}

		// Update song in database
		if *song.LastPerformed == "" {
			song.LastPerformed = nil
		}
		if _, err = db.Exec("UPDATE songs SET artist = $1, location = $2, last_performed = $3, notes = $4, name = $5 WHERE collection_id = $6 AND song_id = $7", song.Artist, song.Location, song.LastPerformed, song.Notes, song.Name, collectionID, song.SongID); err != nil {
			log.Printf("Song PUT - Unable to update song in database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	} else if r.Method == "DELETE" {
		// Delete song
		var result sql.Result
		if result, err = db.Exec("DELETE FROM songs WHERE collection_id = $1 AND song_id = $2", song.CollectionID, song.SongID); err != nil {
			log.Printf("Song DELETE - Unable to delete song from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		var rowsAffected int64
		if rowsAffected, err = result.RowsAffected(); err != nil {
			log.Printf("Song DELETE - Unable to get rows affected. Assuming everything is fine? Error: %v\n", err)
		} else if rowsAffected == 0 {
			log.Printf("Song DELETE - No rows were deleted from the database for song id %d\n", song.SongID)
			SendError(w, `{"error": "No song was found with that ID"}`, http.StatusNotFound)
			return
		}

		log.Printf("Song DELETE - User %d deleted song %d from collection %d.\n", session.Values["user_id"], song.SongID, song.CollectionID)
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
		log.Printf("SongTags handler - Unable to get session: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Get collection ID from URL
	collectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		log.Printf("SongTags handler - Unable to parse collection id from URL: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	// Get song ID from URL
	songID, err = strconv.ParseInt(mux.Vars(r)["song_id"], 10, 64)
	if err != nil {
		log.Printf("SongTags handler - Unable to parse song id from URL: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		// Find the song in the database
		if rows, err = db.Query("SELECT tags.tag_id, name, description FROM tags JOIN tagged_songs ON tags.tag_id = tagged_songs.tag_id WHERE collection_id = $1 AND song_id = $2", collectionID, songID); err != nil {
			if err != sql.ErrNoRows {
				log.Printf("Tagged song GET - Unable to get tagged songs from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			}
			return
		}

		for rows.Next() {
			var tag Tag
			if err = rows.Scan(&tag.TagID, &tag.Name, &tag.Description); err != nil {
				log.Printf("Tagged song GET - Unable to parse tagged song from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			}

			tags = append(tags, tag)
		}

		if err = rows.Err(); err != nil {
			log.Printf("Tagged song GET - Unable to get next tagged song from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
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
			log.Printf("Body: %v\n", r.Body)
			SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
			return
		}

		// Song ID Validation
		var targetCollectionID int64
		if err = db.QueryRow("SELECT collection_id FROM songs WHERE song_id = $1", songID).Scan(&targetCollectionID); err != nil {
			SendError(w, `{"error": "Song not found."}`, http.StatusNotFound)
		}

		if targetCollectionID != collectionID {
			log.Printf("Tagged song POST - User %s (%s) attempted to tag song %d that they didn't own! Error %v\n", session.Values["name"], session.Values["email"], songID, err)
			SendError(w, `{"error": "Song not found."}`, http.StatusNotFound)
			return
		}

		// Tag ID Validation
		if err = db.QueryRow("SELECT collection_id FROM tags WHERE tag_id = $1", taggedSong.TagID).Scan(&targetCollectionID); err != nil {
			SendError(w, `{"error": "Tag not found."}`, http.StatusNotFound)
		}

		if targetCollectionID != collectionID {
			log.Printf("Tagged song POST - User %s (%s) attempted to use tag %d that they didn't own! Error %v\n", session.Values["name"], session.Values["email"], taggedSong.TagID, err)
			SendError(w, `{"error": "Tag not found."}`, http.StatusNotFound)
			return
		}

		// Create song tag in database
		if _, err = db.Exec("INSERT INTO tagged_songs(tag_id, song_id) VALUES ($1, $2)",
			taggedSong.TagID, taggedSong.SongID); err != nil {
			if err.(*pq.Error).Code == "23505" {
				// Song is already tagged with this tag
				SendError(w, `{"error": "Song already has this tag."}`, http.StatusBadRequest)
				return
			}
			log.Printf("Tagged song POST - Unable to add tag to song record in database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
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
			log.Printf("Body: %v\n", r.Body)
			SendError(w, `{"error": "Unable to parse request."}`, http.StatusBadRequest)
			return
		}

		// Song ID Validation
		var targetCollectionID int64
		if err = db.QueryRow("SELECT collection_id FROM songs WHERE song_id = $1", songID).Scan(&targetCollectionID); err != nil {
			SendError(w, `{"error": "Song not found."}`, http.StatusNotFound)
		}

		if targetCollectionID != collectionID {
			log.Printf("Tagged song POST - User %s (%s) attempted to untag song %d that they didn't own! Error %v\n", session.Values["name"], session.Values["email"], songID, err)
			SendError(w, `{"error": "Song not found."}`, http.StatusNotFound)
			return
		}

		// Tag ID Validation
		if err = db.QueryRow("SELECT collection_id FROM tags WHERE tag_id = $1", taggedSong.TagID).Scan(&targetCollectionID); err != nil {
			log.Printf("Tagged song POST - Unable to retreive tag from database: %v\n", err)
			SendError(w, `{"error": "Tag not found."}`, http.StatusNotFound)
		}

		if targetCollectionID != collectionID {
			log.Printf("Tagged song POST - User %s (%s) attempted to untag %d that they didn't own! Error %v\n", session.Values["name"], session.Values["email"], taggedSong.TagID, err)
			SendError(w, `{"error": "Tag not found."}`, http.StatusNotFound)
			return
		}

		// Delete song tag from database
		if _, err = db.Exec("DELETE FROM tagged_songs WHERE tag_id = $1 AND song_id = $2",
			taggedSong.TagID, songID); err != nil {
			log.Printf("Tagged song POST - Unable to delete song tag record from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// All operations completed successfully
		w.WriteHeader(http.StatusOK)
		return
	}
}
