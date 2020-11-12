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
	Email   string `json:"email,omitempty" db:"email"`
	IsAdmin bool   `json:"admin" db:"admin"`
}

// CollectionMembersResponse is a struct for the response to a request for a collection's members
type CollectionMembersResponse struct {
	UserID  int64    `json:"user_id"`
	IsAdmin bool     `json:"admin"`
	Members []Member `json:"members"`
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
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Collections handler - Unable to get session store: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	var collectionID int64

	// Get URL parameter
	collectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		log.Printf("Members handler - Unable to parse collection id: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		var response CollectionMembersResponse

		response.UserID = session.Values["user_id"].(int64)
		response.Members = make([]Member, 0)

		// Get a list of all user's collections
		rows, err := db.Query("SELECT user_id, name, admin FROM collection_members NATURAL JOIN users WHERE collection_id = $1 ORDER BY admin DESC", collectionID)
		if err != nil {
			log.Printf("Members GET - Unable to retrieve collection members from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Sanity check to make sure we've checked the user to be an admin
		var userEncountered bool = false

		// Retrieve rows from database
		for rows.Next() {
			var member Member
			if err := rows.Scan(&member.UserID, &member.Name, &member.IsAdmin); err != nil {
				log.Printf("Unable to retrieve row from database result: %v\n", err)
			}
			response.Members = append(response.Members, member)

			if member.UserID == response.UserID {
				response.IsAdmin = member.IsAdmin
				userEncountered = true
			}
		}

		if !userEncountered {
			log.Printf("Members GET - User not encountered in collection members! Admin value unset. User %d in collection %d\n", response.UserID, collectionID)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Members GET - Error retrieving collection members from database result: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}
}

// CollectionMemberHandler handles removing a user from a collection.
func CollectionMemberHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Collection Member handler - Unable to get session store: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	var collectionID, sourceUserID, targetUserID int64

	// Get URL parameters
	collectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		log.Printf("Collection Member - Unable to parse collection id: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}
	targetUserID, err = strconv.ParseInt(mux.Vars(r)["user_id"], 10, 64)
	if err != nil {
		log.Printf("Collection Member - Unable to parse collection id: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	sourceUserID = session.Values["user_id"].(int64)

	if r.Method == "DELETE" {
		// Check if user is an admin for this collection
		var admin bool
		if err := db.QueryRow("SELECT admin FROM collection_members WHERE user_id = $1 AND collection_id = $2", sourceUserID, collectionID).Scan(&admin); err != nil {
			log.Printf("Collection Member DELETE - Unable to get collection from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		if !admin {
			log.Printf("Collection Member DELETE - User %d attempted to delete user %d from collection %d!\n", sourceUserID, targetUserID, collectionID)
			SendError(w, `{"error": "You do not have permission to perform that action."}`, http.StatusForbidden)
			return
		}

		// Remove target user from collection
		if _, err = db.Exec("DELETE FROM collection_members WHERE user_id = $1 AND collection_id = $2", targetUserID, collectionID); err != nil {
			log.Printf("Collection Member DELETE - Unable to delete collection member: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}
