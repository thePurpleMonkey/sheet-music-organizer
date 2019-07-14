package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// Tag is a struct that models the structure of a tag, both in the request body, and in the DB
type Tag struct {
	Name         string `json:"name", db:"name"`
	Description  string `json:"description", db:"description"`
	CollectionID int    `json:"collection_id", db:"collection_id"`
}

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
		rows, err := db.Query("SELECT name FROM tags WHERE collection_id = $1", collectionID)
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
			if err := rows.Scan(&tag.Name); err != nil {
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

func TagHandler(w http.ResponseWriter, r *http.Request) {}
