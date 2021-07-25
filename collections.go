package main

import (
	"database/sql"
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
	Admin        *bool  `json:"admin,omitempty"`
}

// CollectionsResponse is the data that is returned when the user requests their list of collections
type CollectionsResponse struct {
	UserID      int64        `json:"user_id"`
	Collections []Collection `json:"collections"`
}

// CollectionsHandler handles GETting all of the user's collections and POSTing a new collection.
func CollectionsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Collections handler - Unable to get session store: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		response := CollectionsResponse{
			session.Values["user_id"].(int64),
			make([]Collection, 0),
		}

		// Get a list of all user's collections
		rows, err := db.Query("SELECT collection_id, name, description FROM collection_members NATURAL JOIN collections WHERE user_id = $1", response.UserID)
		if err != nil {
			log.Printf("Collections GET - Unable to retrieve collections from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		for rows.Next() {
			var collection Collection
			if err := rows.Scan(&collection.CollectionID, &collection.Name, &collection.Description); err != nil {
				log.Printf("Unable to retrieve row from database result: %v\n", err)
			}
			response.Collections = append(response.Collections, collection)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Error retrieving collections from database result: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return

	} else if r.Method == "POST" {
		// Parse and decode the request body into a new `Collection` instance
		collection := &Collection{}
		if err := json.NewDecoder(r.Body).Decode(collection); err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Collections POST - Unable to decode request body: %v\n", err)
			SendError(w, `{"error": "Unable to decode request body."}`, http.StatusBadRequest)
			return
		}

		// Input validation
		if collection.Name == "" {
			log.Println("Collections POST - Cannot create a collection with a blank name.")
			SendError(w, `{"error": "Cannot create a collection with a blank name."}`, http.StatusBadRequest)
			return
		}

		// Start db transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Collections POST - Unable to begin database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}

		// Create collection in database
		if err = tx.QueryRow("INSERT INTO collections(name, description) VALUES ($1, $2) RETURNING collection_id", collection.Name, collection.Description).Scan(&collection.CollectionID); err != nil {
			log.Printf("Collections POST - Unable to insert collection into database: %v\n", err)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Collections POST - Unable to rollback transaction: %v", rollbackErr)
			}

			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Add the user as the admin for this collection
		if _, err = tx.Exec("INSERT INTO collection_members VALUES ($1, $2, $3)", session.Values["user_id"], collection.CollectionID, true); err != nil {
			log.Printf("Collections POST - Unable to add user as member of collection: %v\n", err)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Collections POST - Unable to rollback transaction: %v", rollbackErr)
			}
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Save changes
		if err = tx.Commit(); err != nil {
			log.Printf("Collections POST - Unable to commit database transaction: %v\n", err)
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
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Collections handler - Unable to get session store: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	var collection Collection

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
		if err := db.QueryRow("SELECT name, description, admin FROM collections NATURAL JOIN collection_members WHERE collection_id = $1 AND user_id = $2", collection.CollectionID, session.Values["user_id"]).Scan(&collection.Name, &collection.Description, &collection.Admin); err != nil {
			log.Printf("Collection GET - Unable to get collection from database: %v\n", err)
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

		// Verify user is admin of collection
		if admin, err := checkAdmin(session.Values["user_id"].(int64), collection.CollectionID); err != nil {
			log.Printf("Collection PUT - Unable to check admin status for user %d in collection %d: %v\n", session.Values["user_id"], collectionID, err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		} else if !admin {
			log.Printf("Collection PUT - Non-admin user %d attempted to edit collection %d\n", session.Values["user_id"], collectionID)
			SendError(w, PERMISSION_ERROR_MESSAGE, http.StatusForbidden)
			return
		}

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
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return

	} else if r.Method == "DELETE" {
		log.Printf("Collection DELETE - User %d initiated a delete of collection %d\n", session.Values["user_id"], collection.CollectionID)

		// Verify user is admin of collection
		if admin, err := checkAdmin(session.Values["user_id"].(int64), collection.CollectionID); err != nil {
			log.Printf("Collection DELETE - Unable to check admin status for user %d in collection %d: %v\n", session.Values["user_id"], collection.CollectionID, err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		} else if !admin {
			log.Printf("Collection DELETE - Non-admin user %d attempted to delete collection %d\n", session.Values["user_id"], collection.CollectionID)
			SendError(w, PERMISSION_ERROR_MESSAGE, http.StatusForbidden)
			return
		}

		// Start db transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Collection DELETE - Unable to start database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		if err = deleteCollection(collection.CollectionID, tx); err != nil {
			log.Printf("Collection DELETE - Unable to delete collection %d for user %d: %v\n", collection.CollectionID, session.Values["user_id"], err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Save changes
		if err := tx.Commit(); err != nil {
			log.Printf("Collection DELETE - Unable to commit database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		log.Printf("Collection DELETE - User %d successfully deleted collection %d\n", session.Values["user_id"], collection.CollectionID)
		w.WriteHeader(http.StatusOK)
		return
	}
}

func deleteCollection(collectionID int64, tx *sql.Tx) error {
	// Remove tags from songs
	if _, err := tx.Exec("DELETE FROM tagged_songs WHERE song_id IN (SELECT song_id FROM songs WHERE collection_id = $1)", collectionID); err != nil {
		log.Printf("deleteCollection - Unable to delete songs from collection: %v\n", err)
		return err
	}

	// Remove songs from setlists
	if _, err := tx.Exec("DELETE FROM setlist_songs WHERE song_id IN (SELECT song_id FROM songs WHERE collection_id = $1)", collectionID); err != nil {
		log.Printf("deleteCollection - Unable to delete songs from collection: %v\n", err)
		return err
	}

	// Remove setlists
	if _, err := tx.Exec("DELETE FROM setlists WHERE collection_id = $1", collectionID); err != nil {
		log.Printf("deleteCollection - Unable to delete songs from collection: %v\n", err)
		return err
	}

	// Remove songs from collection
	if _, err := tx.Exec("DELETE FROM songs WHERE collection_id = $1", collectionID); err != nil {
		log.Printf("deleteCollection - Unable to delete songs from collection: %v\n", err)
		return err
	}

	// Remove tags from collection
	if _, err := tx.Exec("DELETE FROM tags WHERE collection_id = $1", collectionID); err != nil {
		log.Printf("deleteCollection - Unable to delete tags from collection: %v\n", err)
		return err
	}

	// Remove users from collection
	if _, err := tx.Exec("DELETE FROM collection_members WHERE collection_id = $1", collectionID); err != nil {
		log.Printf("deleteCollection - Unable to delete collection members: %v\n", err)
		return err
	}

	// Remove invitations from collection
	if _, err := tx.Exec("DELETE FROM invitations WHERE collection_id = $1", collectionID); err != nil {
		log.Printf("deleteCollection - Unable to delete invitations: %v\n", err)
		return err
	}

	// Delete collection
	if _, err := tx.Exec("DELETE FROM collections WHERE collection_id = $1", collectionID); err != nil {
		log.Printf("deleteCollection - Unable to delete collection from database: %v\n", err)
		return err
	}

	return nil
}
