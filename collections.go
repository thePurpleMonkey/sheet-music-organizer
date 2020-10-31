package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Collection is a struct that models the structure of a collection, both in the request body, and in the DB
type Collection struct {
	CollectionID int64  `json:"collection_id" db:"collection_id"`
	Name         string `json:"name" db:"name"`
	Description  string `json:"description" db:"description"`
}

// Member is a struct that models the members of a collection
type Member struct {
	UserID  int64  `json:"user_id" db:"user_id"`
	Name    string `json:"name" db:"name"`
	Email   string `json:"email" db:"email"`
	IsAdmin bool   `json:"admin" db:"admin"`
}

// CollectionsHandler handles GETting all of the user's collections and POSTing a new collection.
func CollectionsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Collections handler - Unable to get session store: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		// Get a list of all user's collections
		rows, err := db.Query("SELECT collection_id, name, description FROM collection_members NATURAL JOIN collections WHERE user_id = $1", session.Values["user_id"])
		if err != nil {
			log.Printf("Collections GET - Unable to retrieve collections from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		collections := make([]Collection, 0)
		for rows.Next() {
			var collection Collection
			if err := rows.Scan(&collection.CollectionID, &collection.Name, &collection.Description); err != nil {
				log.Printf("Unable to retrieve row from database result: %v\n", err)
			}
			collections = append(collections, collection)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Error retrieving collections from database result: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
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
			log.Printf("Collections POST - Unable to decode request body: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, `{"error": "Unable to decode request body."}`, http.StatusBadRequest)
			return
		}

		// Input validation
		if collection.Name == "" {
			log.Println("Collections POST - Cannot create a collection with a blank name.")
			w.Header().Add("Content-Type", "application/json")
			SendError(w, `{"error": "Cannot create a collection with a blank name."}`, http.StatusBadRequest)
			return
		}

		// Start db transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Collections POST - Unable to begin database transaction: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}

		// Create collection in database
		if err = tx.QueryRow("INSERT INTO collections(name, description) VALUES ($1, $2) RETURNING collection_id", collection.Name, collection.Description).Scan(&collection.CollectionID); err != nil {
			log.Printf("Collections POST - Unable to insert collection into database: %v\n", err)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Collections POST - Unable to rollback transaction: %v", rollbackErr)
			}

			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Add the user as the admin for this collection
		if _, err = tx.Exec("INSERT INTO collection_members VALUES ($1, $2, $3)", session.Values["user_id"], collection.CollectionID, true); err != nil {
			log.Printf("Collections POST - Unable to add user as member of collection: %v\n", err)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Collections POST - Unable to rollback transaction: %v", rollbackErr)
			}
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Save changes
		if err = tx.Commit(); err != nil {
			log.Printf("Collections POST - Unable to commit database transaction: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		log.Printf("Collections POST - %v | Adding %v to authorized session IDs.", session.Values["email"], collection.CollectionID)
		session.Values["ids"] = append(session.Values["ids"].([]int64), collection.CollectionID)
		if err = session.Save(r, w); err != nil {
			log.Printf("Collections POST - Unable to save session state: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// CollectionHandler handles getting, updating, and deleting a single collection.
func CollectionHandler(w http.ResponseWriter, r *http.Request) {
	// session, err := store.Get(r, "session")
	// if err != nil {
	// 	SendError(w, err.Error(), http.StatusInternalServerError)
	// 	log.Fatal(err)
	// 	return
	// }

	var collection Collection
	var err error

	// Get URL parameter
	collection.CollectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		log.Printf("Collection GET - Unable to parse collection id: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		// Find the collection in the database
		if err := db.QueryRow("SELECT name, description FROM collections WHERE collection_id = $1", collection.CollectionID).Scan(&collection.Name, &collection.Description); err != nil {
			log.Printf("Collection GET - Unable to get collection from database: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
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
			log.Printf("Body: %v\n", r.Body)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, `{"error": "Unable to parse request body."}`, http.StatusBadRequest)
			return
		}

		// Update collection in database
		if _, err = db.Exec("UPDATE collections SET name = $1, description = $2 WHERE collection_id = $3", collection.Name, collection.Description, collectionID); err != nil {
			log.Printf("Collection PUT - Unable to update collection in database: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return

	} else if r.Method == "DELETE" {
		// Start db transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Collection DELETE - Unable to start database transaction: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}

		// Remove songs from collection
		if _, err = tx.Exec("DELETE FROM songs WHERE collection_id = $1", collection.CollectionID); err != nil {
			log.Printf("Collection DELETE - Unable to delete songs from collection: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Remove tags from collection
		if _, err = tx.Exec("DELETE FROM tags WHERE collection_id = $1", collection.CollectionID); err != nil {
			log.Printf("Collection DELETE - Unable to delete tags from collection: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Remove users from collection
		if _, err = tx.Exec("DELETE FROM collection_members WHERE collection_id = $1", collection.CollectionID); err != nil {
			log.Printf("Collection DELETE - Unable to delete collection members: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Delete collection
		if _, err = tx.Exec("DELETE FROM collections WHERE collection_id = $1", collection.CollectionID); err != nil {
			log.Printf("Collection DELETE - Unable to delete collection from database: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Save changes
		if err = tx.Commit(); err != nil {
			log.Printf("Collection DELETE - Unable to commit database transaction: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}

// CollectionMembersHandler handles getting, adding, and deleting members from a collection.
func CollectionMembersHandler(w http.ResponseWriter, r *http.Request) {
	// session, err := store.Get(r, "session")
	// if err != nil {
	// 	SendError(w, err.Error(), http.StatusInternalServerError)
	// 	log.Fatal(err)
	// 	return
	// }

	//var collection Collection
	var collectionID int64
	var err error

	// Get URL parameter
	collectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		log.Printf("Members handler - Unable to parse collection id: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		// Get a list of all user's collections
		rows, err := db.Query("SELECT name, admin FROM collection_members NATURAL JOIN users WHERE collection_id = $1 ORDER BY admin DESC", collectionID)
		if err != nil {
			log.Printf("Members GET - Unable to retrieve collection members from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		members := make([]Member, 0)
		for rows.Next() {
			var member Member
			if err := rows.Scan(&member.Name, &member.IsAdmin); err != nil {
				log.Printf("Unable to retrieve row from database result: %v\n", err)
			}
			members = append(members, member)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Error retrieving collection members from database result: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(members)
		return

	} else if r.Method == "POST" {
		log.Printf("Members POST - Not implemented.")
		SendError(w, "Not Found", http.StatusNotFound)
		return

		// var member Member
		// err := json.NewDecoder(r.Body).Decode(&member)
		// if err != nil {
		// 	// If there is something wrong with the request body, return a 400 status
		// 	log.Printf("Member POST - Unable to parse request body: %v\n", err)
		// 	log.Printf("Body: %v\n", r.Body)
		// 	w.Header().Add("Content-Type", "application/json")
		// 	SendError(w, `{"error": "Unable to parse request body."}`, http.StatusBadRequest)
		// 	return
		// }

		// // Update collection in database
		// if _, err = db.Exec("UPDATE collections SET name = $1, description = $2 WHERE collection_id = $3", collection.Name, collection.Description, collectionID); err != nil {
		// 	log.Printf("Collection PUT - Unable to update collection in database: %v\n", err)
		// 	w.Header().Add("Content-Type", "application/json")
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// w.WriteHeader(http.StatusOK)
		// return

	} else if r.Method == "DELETE" {
		log.Printf("Members DELETE - Not implemented.")
		SendError(w, "Not Found", http.StatusNotFound)
		return

		// // Start db transaction
		// tx, err := db.Begin()
		// if err != nil {
		// 	log.Printf("Collection DELETE - Unable to start database transaction: %v\n", err)
		// 	w.Header().Add("Content-Type", "application/json")
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// }

		// // Remove songs from collection
		// if _, err = tx.Exec("DELETE FROM songs WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	log.Printf("Collection DELETE - Unable to delete songs from collection: %v\n", err)
		// 	w.Header().Add("Content-Type", "application/json")
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// // Remove tags from collection
		// if _, err = tx.Exec("DELETE FROM tags WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	log.Printf("Collection DELETE - Unable to delete tags from collection: %v\n", err)
		// 	w.Header().Add("Content-Type", "application/json")
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// // Remove users from collection
		// if _, err = tx.Exec("DELETE FROM collection_members WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	log.Printf("Collection DELETE - Unable to delete collection members: %v\n", err)
		// 	w.Header().Add("Content-Type", "application/json")
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// // Delete collection
		// if _, err = tx.Exec("DELETE FROM collections WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	log.Printf("Collection DELETE - Unable to delete collection from database: %v\n", err)
		// 	w.Header().Add("Content-Type", "application/json")
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// // Save changes
		// if err = tx.Commit(); err != nil {
		// 	log.Printf("Collection DELETE - Unable to commit database transaction: %v\n", err)
		// 	w.Header().Add("Content-Type", "application/json")
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// w.WriteHeader(http.StatusOK)
		// return
	}
}
