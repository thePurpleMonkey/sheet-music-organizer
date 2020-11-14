package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Member is a struct that models the members of a collection
type Member struct {
	UserID  int64  `json:"user_id" db:"user_id"`
	Name    string `json:"name" db:"name"`
	Email   string `json:"email,omitempty" db:"email"`
	IsAdmin bool   `json:"admin" db:"admin"`
}

// MembersResponse is a struct for the response to a request for a collection's members
type MembersResponse struct {
	UserID  int64    `json:"user_id"`
	IsAdmin bool     `json:"admin"`
	Members []Member `json:"members"`
}

// MembersHandler handles getting, adding, and deleting members from a collection.
func MembersHandler(w http.ResponseWriter, r *http.Request) {
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
		var response MembersResponse

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

// MemberHandler handles removing a user from a collection.
func MemberHandler(w http.ResponseWriter, r *http.Request) {
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
