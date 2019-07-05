package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// Collection is a struct that models the structure of a collection, both in the request body, and in the DB
type Collection struct {
	CollectionID int64  `json:"collection_id", db:"collection_id"`
	Name         string `json:"name", db:"name"`
	Description  string `json:"description", db:"description"`
}

func CollectionsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
		return
	}

	if r.Method == "GET" {
		// Get a list of all user's collections
		rows, err := db.Query("SELECT collection_id, name, description FROM collection_members NATURAL JOIN collections WHERE user_email = $1", session.Values["email"])
		if err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error retrieving collections from database."}`))
			log.Fatal(err)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		collections := make([]Collection, 0)
		for rows.Next() {
			var collection Collection
			if err := rows.Scan(&collection.CollectionID, &collection.Name, &collection.Description); err != nil {
				log.Fatal(err)
			}
			collections = append(collections, collection)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error retrieving collections from database."}`))
			log.Fatal(err)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(collections)
		return

	} else if r.Method == "POST" {
		// Parse and decode the request body into a new `Collection` instance
		collection := &Collection{}
		if err := json.NewDecoder(r.Body).Decode(collection); err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Println(err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Start db transaction
		tx, err := db.Begin()
		if err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Unable to begin database transaction."}`))
		}

		// Create collection in database
		if err = tx.QueryRow("INSERT INTO collections(name, description) VALUES ($1, $2) RETURNING collection_id", collection.Name, collection.Description).Scan(&collection.CollectionID); err != nil {
			log.Println(err)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Fatalf("Unable to rollback transaction: %v", rollbackErr)
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Unable to create collection."}`))
			return
		}

		// Add the user as the admin for this collection
		if _, err = tx.Exec("INSERT INTO collection_members VALUES ($1, $2, $3)", session.Values["email"], collection.CollectionID, true); err != nil {
			log.Println(err)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Fatalf("Unable to rollback transaction: %v", rollbackErr)
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Unable to add admin to collection."}`))
			return
		}

		// Save changes
		if err = tx.Commit(); err != nil {
			log.Println(err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Unable to commit transaction to database."}`))
			return
		}
	}
}

func CollectionHandler(w http.ResponseWriter, r *http.Request) {
	// session, err := store.Get(r, "session")
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	log.Fatal(err)
	// 	return
	// }

	if r.Method == "GET" {
	}
}
