package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// Date is a struct that allows custom unmarshalling behavior for dates
type Date struct {
	*time.Time
}

// UnmarshalJSON sets time to nil for empty string, otherwise passes through to json.Unmarshal
func (d *Date) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte(`""`)) {
		d.Time = nil
		return nil
	}

	return json.Unmarshal(b, &d.Time)
}

// MarshalJSON marshals the custom Date object for encoding
func (d *Date) MarshalJSON() ([]byte, error) {
	if d.Time == nil {
		return []byte("null"), nil
	}

	return json.Marshal(d.Time)
}

// Song is a struct that models the structure of a song, both in the request body, and in the DB
type Song struct {
	Name          string `json:"name", db:"name"`
	Artist        string `json:"artist", db:"artist"`
	DateAdded     Date   `json:"date_added", db:"date_added"`
	Location      string `json:"location", db:"location"`
	LastPerformed Date   `json:"last_performed", db:"last_performed"`
	Notes         string `json:"notes", db:"notes"`
	AddedBy       int    `json:"added_by", db:"added_by"`
	CollectionID  string `json:"collection_id", db:"collection_id"`
}

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
		rows, err := db.Query("SELECT name FROM songs WHERE collection_id = $1", collectionID)
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
			if err := rows.Scan(&song.Name); err != nil {
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
			song.Name, song.Artist, song.Location, song.LastPerformed.Time, song.Notes, session.Values["email"], collectionID); err != nil {
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

func SongHandler(w http.ResponseWriter, r *http.Request) {}
