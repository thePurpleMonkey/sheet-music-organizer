package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

var DATABASE_ERROR_MESSAGE = []byte(`{"error": "Error communicating with database."}`)

// Collection is a struct that models the structure of a collection, both in the request body, and in the DB
type Collection struct {
	CollectionID int64  `json:"collection_id" db:"collection_id"`
	Name         string `json:"name" db:"name"`
	Description  string `json:"description" db:"description"`
}

// CollectionsHandler handles GETting all of the user's collections and POSTing a new collection.
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
			log.Printf("Collections GET - Unable to retrieve collections from database: %v\n", err)
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
			log.Printf("Error retrieving collections from database: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(DATABASE_ERROR_MESSAGE)
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
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Unable to decode request body."}`))
			log.Printf("Collections POST - Unable to decode request body: %v\n", err)
			return
		}

		// Input validation
		if collection.Name == "" {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Cannot create a collection with a blank name."}`))
			log.Println("Collections POST - Cannot create a collection with a blank name.")
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

		log.Printf("%v | Adding %v to authorized session IDs.", session.Values["email"], collection.CollectionID)
		session.Values["ids"] = append(session.Values["ids"].([]int64), collection.CollectionID)
		if err = session.Save(r, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// CollectionHandler handles getting, updating, and deleting a single collection.
func CollectionHandler(w http.ResponseWriter, r *http.Request) {
	// session, err := store.Get(r, "session")
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	log.Fatal(err)
	// 	return
	// }

	var collection Collection
	var err error

	// Get URL parameter
	collection.CollectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unable to parse collection id."}`))
		log.Printf("Collection GET - Unable to parse collection id: %v\n", err)
		return
	}

	if r.Method == "GET" {
		// Find the collection in the database
		if err := db.QueryRow("SELECT name, description FROM collections WHERE collection_id = $1", collection.CollectionID).Scan(&collection.Name, &collection.Description); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error retrieving collection from database."}`))
			log.Printf("Collection GET - Unable to get collection from database: %v\n", err)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(collection)
		return

	} else if r.Method == "PUT" {
		// Save the URL collection ID so the user can't update another record
		var collectionID = collection.CollectionID

		err := json.NewDecoder(r.Body).Decode(&collection)
		if err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Collection PUT - Unable to parse request body: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Update collection in database
		if _, err = db.Exec("UPDATE collections SET name = $1, description = $2 WHERE collection_id = $3", collection.Name, collection.Description, collectionID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Collection PUT - Unable to update collection in database: %v\n", err)
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
			log.Printf("Collection DELETE - Unable to start database transaction: %v\n", err)
		}

		// Remove songs from collection
		if _, err = tx.Exec("DELETE FROM songs WHERE collection_id = $1", collection.CollectionID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Collection DELETE - Unable to delete songs from collection: %v\n", err)
			return
		}

		// Remove tags from collection
		if _, err = tx.Exec("DELETE FROM tags WHERE collection_id = $1", collection.CollectionID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Collection DELETE - Unable to delete tags from collection: %v\n", err)
			return
		}

		// Remove users from collection
		if _, err = tx.Exec("DELETE FROM collection_members WHERE collection_id = $1", collection.CollectionID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Collection DELETE - Unable to delete collection members: %v\n", err)
			return
		}

		// Delete collection
		if _, err = tx.Exec("DELETE FROM collections WHERE collection_id = $1", collection.CollectionID); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Collection DELETE - Unable to delete collection from database: %v\n", err)
			return
		}

		// Save changes
		if err = tx.Commit(); err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(DATABASE_ERROR_MESSAGE))
			log.Printf("Collection DELETE - Unable to commit database transaction: %v\n", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}
