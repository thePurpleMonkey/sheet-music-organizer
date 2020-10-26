package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// Tag is a struct that models the structure of a tag, both in the request body, and in the DB
type Tag struct {
	TagID        int64  `json:"tag_id" db:"tag_id"`
	Name         string `json:"name" db:"name"`
	Description  string `json:"description" db:"description"`
	CollectionID int64  `json:"collection_id" db:"collection_id"`
}

// TagsHandler handles GETting all tags or POSTing a new tag.
func TagsHandler(w http.ResponseWriter, r *http.Request) {
	// session, err := store.Get(r, "session")
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	log.Fatalf("Unable to get session: %v\n", err)
	// 	return
	// }

	// Get URL parameter
	var collectionID int
	collectionID, err := strconv.Atoi(mux.Vars(r)["collection_id"])
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse collection id."}`))
		log.Println(err)
		return
	}

	if r.Method == "GET" {
		// Retrieve Tags in collection
		rows, err := db.Query("SELECT tag_id, name, description FROM tags WHERE collection_id = $1", collectionID)
		if err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error retrieving tags from database."}`))
			log.Println(err)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		tags := make([]Tag, 0)
		for rows.Next() {
			var tag Tag
			if err := rows.Scan(&tag.TagID, &tag.Name, &tag.Description); err != nil {
				log.Fatal(err)
			}
			tags = append(tags, tag)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error retrieving tags from database."}`))
			log.Fatal(err)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(tags)
		return

	} else if r.Method == "POST" {
		// Parse and decode the request body into a new `Tag` instance
		tag := &Tag{}
		if err := json.NewDecoder(r.Body).Decode(tag); err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Unable to decode request body in TagsHandler (POST): %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Unable to decode request body."}`))
			return
		}

		// Input validation
		if len(tag.Name) == 0 {
			log.Println("Invalid Tag Name in TagsHandler POST request.")
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "No tag name supplied."}`))
			return
		}

		// Create collection in database
		if _, err = db.Exec("INSERT INTO tags(name, description, collection_id) VALUES ($1, $2, $3)",
			tag.Name, tag.Description, collectionID); err != nil {
			if err.(*pq.Error).Code == "23505" {
				// Tag already exists
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Tag already exists."}`))
				return
			}
			log.Printf("Unable to insert tag record in database: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Unable to add tag."}`))
			return
		}

		// Respond with success
		w.WriteHeader(http.StatusCreated)
	}
}

// TagHandler handles creating, updating, or deleting a single tag.
func TagHandler(w http.ResponseWriter, r *http.Request) {
	var tag Tag
	var err error

	// Get collection ID from URL
	tag.CollectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse URL."}`))
		log.Printf("Tag handler - Unable to parse collection id from URL: %v\n", err)
		return
	}

	// Get tag ID from URL
	tag.TagID, err = strconv.ParseInt(mux.Vars(r)["tag_id"], 10, 64)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse tag id."}`))
		log.Println(err)
		return
	}

	if r.Method == "GET" {
		// Find the tag in the database
		if err = db.QueryRow("SELECT tag_id, name, description FROM tags WHERE collection_id = $1 AND tags.tag_id = $2", tag.CollectionID, tag.TagID).Scan(&tag.TagID, &tag.Name, &tag.Description); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Tag GET - No tag found for collection %v and tag id %v\n", tag.CollectionID, tag.TagID)
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(DATABASE_ERROR_MESSAGE)
				log.Printf("Tag GET - Unable to get tag from database: %v\n", err)
			}
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(tag)
		return

	} else if r.Method == "PUT" {
		// Save the URL collection ID so the user can't update another record
		var collectionID = tag.CollectionID

		err := json.NewDecoder(r.Body).Decode(&tag)
		if err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Tag PUT - Unable to parse request body: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Unable to parse request."}`))
			return
		}

		// Update tag in database
		var result sql.Result
		if result, err = db.Exec("UPDATE tags SET name = $1, description = $2 WHERE collection_id = $3 AND tag_id = $4", tag.Name, tag.Description, collectionID, tag.TagID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Tag PUT - Unable to update tag in database: %v\n", err)
			return
		}

		// Check if update did anything
		var rows int64
		rows, err = result.RowsAffected()
		if err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Tag PUT - Database update unsuccessful: %v\n", err)
			return
		}

		if rows == 0 {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Tag not found."}`))
			log.Printf("Tag PUT - Tag id '%v' not found in database: %v\n", tag.TagID, err)
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
			log.Printf("Tag DELETE - Unable to start database transaction: %v\n", err)
		}

		// Removed tagged songs
		if _, err = tx.Exec("DELETE FROM tagged_songs WHERE tag_id = $1", tag.TagID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Tag DELETE - Unable to remove tagged songs from database: %v\n", err)
			return
		}

		// Delete tag
		if _, err = tx.Exec("DELETE FROM tags WHERE collection_id = $1 AND tag_id = $2", tag.CollectionID, tag.TagID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Tag DELETE - Unable to delete tag from database: %v\n", err)
			return
		}

		// Save changes
		if err = tx.Commit(); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Tag DELETE - Unable to commit database transaction: %v\n", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}

// TagSongsHandler handles associating and disassociating a tag with a song.
func TagSongsHandler(w http.ResponseWriter, r *http.Request) {
	var tag Tag
	var err error

	session, err := store.Get(r, "session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatalf("Unable to get session: %v\n", err)
		return
	}

	// Get collection ID from URL
	tag.CollectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse URL."}`))
		log.Printf("Tag handler - Unable to parse collection id from URL: %v\n", err)
		return
	}

	// Get tag ID from URL
	tag.TagID, err = strconv.ParseInt(mux.Vars(r)["tag_id"], 10, 64)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse tag id."}`))
		log.Println(err)
		return
	}

	// Verify Tag ID
	var targetCollectionID int64
	if err = db.QueryRow("SELECT collection_id FROM tags WHERE tag_id = $1", tag.TagID).Scan(&targetCollectionID); err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Tag not found."}`))
	}

	if targetCollectionID != tag.CollectionID {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Tag not found."}`))
		log.Printf("Tagged song POST - User %s (%s) attempted to access tag %d that they didn't own! Error %v\n", session.Values["name"], session.Values["email"], tag.TagID, err)
		return
	}

	if r.Method == "GET" {
		// Retrieve Tags in collection
		rows, err := db.Query("SELECT songs.song_id, songs.name FROM songs JOIN tagged_songs ON songs.song_id = tagged_songs.song_id WHERE collection_id = $1 AND tag_id = $2", tag.CollectionID, tag.TagID)
		if err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error retrieving tags from database."}`))
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

	}
}
