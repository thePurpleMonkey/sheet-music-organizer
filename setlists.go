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

// Setlist is a struct that models the structure of a setlist, both in the request body, and in the DB
type Setlist struct {
	SetlistID int64      `json:"setlist_id"`
	Name      string     `json:"name"`
	Date      *time.Time `json:"date,omitempty"`
	Notes     string     `json:"notes,omitempty"`
}

// ReorderRequest is a struct that modes a request to reorder a setlist
type ReorderRequest struct {
	SongID int64 `json:"song_id"`
	Order  int64 `json:"order"`
}

// SetlistsHandler handles GETting all of the user's setlists and POSTing a new setlist.
func SetlistsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Setlists handler - Unable to get session store: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Get URL parameter
	var collectionID int
	collectionID, err = strconv.Atoi(mux.Vars(r)["collection_id"])
	if err != nil {
		log.Printf("Setlists Handler - Unable to parse collection ID: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		// Get a list of all user's setlists in this collection
		rows, err := db.Query("SELECT setlist_id, name, date, notes FROM setlists WHERE user_id = $1 AND collection_id = $2", session.Values["user_id"], collectionID)
		if err != nil {
			log.Printf("Setlists GET - Unable to retrieve setlists from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		setlists := make([]Setlist, 0)
		for rows.Next() {
			var setlist Setlist
			if err := rows.Scan(&setlist.SetlistID, &setlist.Name, &setlist.Date, &setlist.Notes); err != nil {
				log.Printf("Unable to retrieve row from database result: %v\n", err)
			}
			setlists = append(setlists, setlist)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Error retrieving setlists from database result: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(setlists)
		return

	} else if r.Method == "POST" {
		// Parse and decode the request body into a new `Setlist` instance
		setlist := &Setlist{}
		if err := json.NewDecoder(r.Body).Decode(setlist); err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Setlists POST - Unable to decode request body: %v\n", err)
			body, _ := ioutil.ReadAll(r.Body)
			log.Printf("Body: %s\n", body)
			SendError(w, REQUEST_ERROR_MESSAGE, http.StatusBadRequest)
			return
		}

		// Input validation
		if setlist.Name == "" {
			log.Println("Setlists POST - Cannot create a setlist with a blank name.")
			SendError(w, `{"error": "Cannot create a setlist with a blank name."}`, http.StatusBadRequest)
			return
		}

		// Create setlist in database
		if err = db.QueryRow("INSERT INTO setlists(name, date, notes, collection_id, user_id) VALUES ($1, $2, $3, $4, $5) RETURNING setlist_id", setlist.Name, setlist.Date, setlist.Notes, collectionID, session.Values["user_id"]).Scan(&setlist.SetlistID); err != nil {
			log.Printf("Setlists POST - Unable to insert setlist into database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// SetlistHandler handles getting, updating, and deleting a single setlist.
func SetlistHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Setlists handler - Unable to get session store: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	var setlist Setlist

	// Get URL parameters
	setlistID, err := strconv.ParseInt(mux.Vars(r)["setlist_id"], 10, 64)
	if err != nil {
		log.Printf("Setlist Handler - Unable to parse setlist ID: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	collectionID, err := strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		log.Printf("Setlists Handler - Unable to parse collection ID: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		// Find the setlist in the database
		if err := db.QueryRow("SELECT setlist_id, name, date, notes FROM setlists WHERE setlist_id = $1 AND user_id = $2", setlistID, session.Values["user_id"]).Scan(&setlist.SetlistID, &setlist.Name, &setlist.Date, &setlist.Notes); err != nil {
			log.Printf("Setlist GET - Unable to get setlist from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(setlist)
		return

	} else if r.Method == "PUT" {
		err := json.NewDecoder(r.Body).Decode(&setlist)
		if err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Setlist PUT - Unable to parse request body: %v\n", err)
			body, _ := ioutil.ReadAll(r.Body)
			log.Printf("Body: %s\n", body)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, REQUEST_ERROR_MESSAGE, http.StatusBadRequest)
			return
		}

		// Update setlist in database
		var result sql.Result
		// log.Printf("SetlistID: %d\tCollectionID: %d\tUserID: %d\n", setlistID, collectionID, session.Values["user_id"])
		// log.Printf("Name: %s\tDate: %s\tNotes: %s\n", setlist.Name, setlist.Date, setlist.Notes)
		if result, err = db.Exec("UPDATE setlists SET name = $1, date = $2, notes = $3 WHERE setlist_id = $4 AND collection_id = $5 AND user_id = $6", setlist.Name, setlist.Date, setlist.Notes, setlistID, collectionID, session.Values["user_id"]); err != nil {
			log.Printf("Setlist PUT - Unable to update setlist in database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("Setlist PUT - Unable to get rows affected by UPDATE: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			log.Printf("Setlist PUT - No rows updated for UPDATE query.\n")
			SendError(w, `{"error": "Setlist not found."}`, http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		return

	} else if r.Method == "DELETE" {
		// Start db transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Setlist DELETE - Unable to start database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Find the setlist in the database
		var actualCollectionID int64
		if err := tx.QueryRow("SELECT collection_id FROM setlists WHERE setlist_id = $1 AND user_id = $2", setlistID, session.Values["user_id"]).Scan(&actualCollectionID); err != nil {
			log.Printf("Setlist DELETE - Unable to get setlist from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		if actualCollectionID != collectionID {
			log.Printf("Setlist DELETE - User %v requested deletion of setlist %d in collection %d, setlist is in collection %d.\n", session.Values["user_id"], setlistID, collectionID, actualCollectionID)
			SendError(w, `{"error": "Setlist not found."}`, http.StatusNotFound)
			return
		}

		err = deleteSetlist(setlistID, tx)

		if err != nil {
			log.Printf("Setlist DELETE - Unable to delete setlist: %v\n", err)
			if txErr := tx.Rollback(); txErr != nil {
				log.Printf("Setlist DELETE - Unable to rollback database transaction: %v\n", txErr)
			}
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Save changes
		if err := tx.Commit(); err != nil {
			log.Printf("Setlist DELETE - Unable to commit database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}

// SetlistSongsHandler handles associating and disassociating a setlist with one or more songs.
func SetlistSongsHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("SetlistSongs handler - Unable to get session: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Get collection ID from URL
	collectionID, err := strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		log.Printf("SetlistSongs handler - Unable to parse collection id from URL: %v\n", err)
		SendError(w, `{"error": "Unable to parse URL."}`, http.StatusBadRequest)
		return
	}

	// Get setlist ID from URL
	setlistID, err := strconv.ParseInt(mux.Vars(r)["setlist_id"], 10, 64)
	if err != nil {
		log.Printf("SetlistSongs handler - Unable to parse setlist_id from URL: %v\n", err)
		SendError(w, `{"error": "Unable to parse URL."}`, http.StatusBadRequest)
		return
	}

	// Verify Setlist ID
	var targetCollectionID int64
	if err = db.QueryRow("SELECT collection_id FROM setlists WHERE setlist_id = $1", setlistID).Scan(&targetCollectionID); err != nil {
		SendError(w, `{"error": "Setlist not found."}`, http.StatusNotFound)
	}

	if targetCollectionID != collectionID {
		log.Printf("SetlistSongs handler - User %s (%s) attempted to access setlist %d that they didn't own! Error %v\n", session.Values["name"], session.Values["email"], setlistID, err)
		SendError(w, `{"error": "Setlist not found."}`, http.StatusNotFound)
		return
	}

	if r.Method == "GET" {
		// Retrieve songs in setlist
		rows, err := db.Query("SELECT songs.song_id, songs.name, songs.date_added, setlist_songs.order FROM songs JOIN setlist_songs ON songs.song_id = setlist_songs.song_id WHERE collection_id = $1 AND setlist_id = $2", collectionID, setlistID)
		if err != nil {
			log.Printf("SetlistSongs GET - Unable to get songs in setlist %v from database: %v\n", setlistID, err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		songs := make([]Song, 0)
		for rows.Next() {
			var song Song
			if err := rows.Scan(&song.SongID, &song.Name, &song.DateAdded, &song.Order); err != nil {
				log.Printf("SetlistSongs GET - Unable to get song data from database result: %v\n", err)
			}
			songs = append(songs, song)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("SetlistSongs GET - Unable to get songs from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(songs)
		return

	} else if r.Method == "POST" {
		// Parse and decode the request body into a new `Setlist` instance
		songs := []int64{}
		if err := json.NewDecoder(r.Body).Decode(&songs); err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Setlists Songs POST - Unable to decode request body: %v\n", err)
			body, _ := ioutil.ReadAll(r.Body)
			log.Printf("Body: %s\n", body)
			SendError(w, REQUEST_ERROR_MESSAGE, http.StatusBadRequest)
			return
		}

		// Input validation
		if len(songs) == 0 {
			log.Println("Setlists Songs POST - Empty song list.")
			SendError(w, `{"error": "You must provide a list of at least 1 song."}`, http.StatusBadRequest)
			return
		}

		// Start database transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Setlists Songs POST - Unable to begin transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		stmt, err := tx.Prepare(pq.CopyIn("setlist_songs", "setlist_id", "song_id"))
		if err != nil {
			log.Printf("Setlists Songs POST - Unable to prepare statement: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		for _, songID := range songs {
			_, err = stmt.Exec(setlistID, songID)
			if err != nil {
				log.Printf("Setlists Songs POST - Unable to add song ID to prepared statement: %v\n", err)
			}
		}

		_, err = stmt.Exec()
		if err != nil {
			log.Printf("Setlists Songs POST - Unable to execute prepared statement: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		err = stmt.Close()
		if err != nil {
			log.Printf("Setlists Songs POST - Unable to close prepared statement: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		err = tx.Commit()
		if err != nil {
			log.Printf("Setlists Songs POST - Unable to commit database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	} else if r.Method == "PUT" {
		var songs []ReorderRequest

		if err := json.NewDecoder(r.Body).Decode(&songs); err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Setlists Songs PUT - Unable to decode request body: %v\n", err)
			SendError(w, REQUEST_ERROR_MESSAGE, http.StatusBadRequest)
			return
		}

		// Input validation
		if len(songs) == 0 {
			log.Println("Setlists Songs PUT - Empty song list.")
			SendError(w, `{"error": "You must provide a list of at least 1 song."}`, http.StatusBadRequest)
			return
		}

		log.Printf("%v\n", songs)

		// Start database transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Setlists Songs PUT - Unable to begin transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Prepare bulk statement
		stmt, err := tx.Prepare(`UPDATE setlist_songs SET "order" = $1 WHERE setlist_id = $2 AND song_id = $3`)
		if err != nil {
			log.Printf("Setlists Songs PUT - Unable to prepare statement: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		for _, order := range songs {
			_, err = stmt.Exec(order.Order, setlistID, order.SongID)
			if err != nil {
				log.Printf("Setlists Songs PUT - Unable to add song ID and order to prepared statement: %v\n", err)
			}
		}

		// _, err = stmt.Exec()
		// if err != nil {
		// 	log.Printf("Setlists Songs PUT - Unable to execute prepared statement: %v\n", err)
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		err = stmt.Close()
		if err != nil {
			log.Printf("Setlists Songs PUT - Unable to close prepared statement: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		err = tx.Commit()
		if err != nil {
			log.Printf("Setlists Songs PUT - Unable to commit database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// SetlistSongHandler manages a single song within a setlist
func SetlistSongHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Setlist Song handler - Unable to get session: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Get collection ID from URL
	collectionID, err := strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		log.Printf("Setlist Song handler - Unable to parse collection id from URL: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	// Get setlist ID from URL
	setlistID, err := strconv.ParseInt(mux.Vars(r)["setlist_id"], 10, 64)
	if err != nil {
		log.Printf("Setlist Song handler - Unable to parse setlist_id from URL: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	// Get song ID from URL
	songID, err := strconv.ParseInt(mux.Vars(r)["song_id"], 10, 64)
	if err != nil {
		log.Printf("Setlist Song handler - Unable to parse song_id from URL: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	// Verify Setlist ID
	var actualCollectionID int64
	if err = db.QueryRow("SELECT collection_id FROM setlists WHERE setlist_id = $1", setlistID).Scan(&actualCollectionID); err != nil {
		log.Printf("Setlist Song handler - Unable to get setlist collection_id from database: %v\n", err)
		SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	if actualCollectionID != collectionID {
		log.Printf("Setlist Song handler - User %s (%s) attempted to modify setlist %d in a different collection!\n", session.Values["name"], session.Values["email"], setlistID)
		SendError(w, `{"error": "Setlist not found."}`, http.StatusNotFound)
		return
	}

	if r.Method == "DELETE" {
		// Delete setlist song from database
		var result sql.Result
		if result, err = db.Exec("DELETE FROM setlist_songs WHERE setlist_id = $1 AND song_id = $2", setlistID, songID); err != nil {
			log.Printf("Setlist Song DELETE - Unable to remove song from setlist: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		var rowsAffected int64
		rowsAffected, err = result.RowsAffected()

		if err != nil {
			log.Printf("Setlist Song DELETE - Unable to get rows affected by delete: %v\n", err)
			// I guess just ignore this and continue then?
		} else if rowsAffected == 0 {
			log.Printf("Setlist Song DELETE - No rows were deleted from the database.\n")
			SendError(w, `{"error": "Setlist song not found."}`, http.StatusNotFound)
			return
		}

		// All operations completed successfully
		w.WriteHeader(http.StatusOK)
		return
	}
}

func deleteSetlist(setlistID int64, tx *sql.Tx) error {
	// Remove songs from setlist
	if _, err := tx.Exec("DELETE FROM setlist_songs WHERE setlist_id = $1", setlistID); err != nil {
		log.Printf("deleteSetlist - Unable to delete songs from setlist: %v\n", err)
		return err
	}

	// Delete setlist
	if _, err := tx.Exec("DELETE FROM setlists WHERE setlist_id = $1", setlistID); err != nil {
		log.Printf("deleteSetlist - Unable to delete setlist: %v\n", err)
		return err
	}

	return nil
}
