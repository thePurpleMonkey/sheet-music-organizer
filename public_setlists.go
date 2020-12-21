package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// PublicSetlist is a struct that models the structure of a setlist, both in the request body, and in the DB
type PublicSetlist struct {
	Name      string     `json:"name"`
	Date      *time.Time `json:"date,omitempty"`
	Notes     string     `json:"notes,omitempty"`
	ShareCode *string    `json:"share_code,omitempty"`
}

// PublicSong is a struct that models the public structure of a song
type PublicSong struct {
	Name  string `json:"name" db:"name"`
	Order int64  `json:"order,omitempty" db:"order"`
}

// PublicSetlistHandler handles getting the public version of a setlist
func PublicSetlistHandler(w http.ResponseWriter, r *http.Request) {
	var setlist PublicSetlist

	// Get URL parameters
	shareCode := mux.Vars(r)["share_code"]

	// if err != nil {
	// 	log.Printf("Setlist Handler - Unable to parse setlist ID: %v\n", err)
	// 	w.Header().Add("Content-Type", "application/json")
	// 	SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
	// 	return
	// }

	if r.Method == "GET" {
		// Find the setlist in the database
		if err := db.QueryRow("SELECT name, date, notes FROM setlists WHERE share_code = $1 AND shared = true", shareCode).Scan(&setlist.Name, &setlist.Date, &setlist.Notes); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Public Setlist GET - No setlist found with share code '%v'\n", shareCode)
				SendError(w, `{"error": "Setlist not found"}`, http.StatusNotFound)
				return
			}
			log.Printf("Public Setlist GET - Unable to get setlist from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(setlist)
		return

	}
}

// PublicSetlistSongsHandler handles returning songs from a public setlist
func PublicSetlistSongsHandler(w http.ResponseWriter, r *http.Request) {
	// Get URL parameters
	shareCode := mux.Vars(r)["share_code"])

	if r.Method == "GET" {
		// Retrieve songs in setlist
		rows, err := db.Query(`
		SELECT songs.name, setlist_songs.order
		FROM songs 
		JOIN setlist_songs ON songs.song_id = setlist_songs.song_id
		JOIN setlists ON setlist_songs.setlist_id = setlists.setlist_id
		WHERE setlists.share_code = $1
		  AND setlists.shared = true`, shareCode)
		if err != nil {
			log.Printf("Public Setlist Songs GET - Unable to get songs in setlist %v from database: %v\n", shareCode, err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		songs := make([]PublicSong, 0)
		for rows.Next() {
			var song PublicSong
			if err := rows.Scan(&song.Name, &song.Order); err != nil {
				log.Printf("Public Setlist Songs GET - Unable to get song data from database result: %v\n", err)
			}
			songs = append(songs, song)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Public Setlist Songs GET - Unable to get songs from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(songs)
		return

	}
}
