package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Invite is a struct that models the invitations to a collection
type Invite struct {
	InvitationID int64      `json:"invitation_id" db:"invitation_id"`
	InviteeEmail string     `json:"invitee_email" db:"invitee_email"`
	AdminInvite  bool       `json:"admin_invite" db:"admin_invite"`
	InviteSent   *time.Time `json:"invite_sent" db:"invite_sent"`
}

// CollectionInvitationsHandler handles inviting new members, managing invitations, and revoking invitations to a collection.
func CollectionInvitationsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		log.Printf("Invitations handler - Unable to get session: %v\n", err)
		return
	}

	//var collection Collection
	var collectionID int64

	// Get URL parameter
	collectionID, err = strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
	if err != nil {
		log.Printf("Invitations handler - Unable to parse collection id: %v\n", err)
		SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		// Get a list of all user's collections
		rows, err := db.Query("SELECT invitation_id, invitee_email, admin_invite, invite_sent FROM invitations WHERE inviter_id = $1 AND collection_id = $2", session.Values["user_id"], collectionID)
		if err != nil {
			log.Printf("Invitations GET - Unable to retrieve invitations from database for user %v: %v\n", session.Values["user_id"], err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		invitations := make([]Invite, 0)
		for rows.Next() {
			var invite Invite
			if err := rows.Scan(&invite.InvitationID, &invite.InviteeEmail, &invite.AdminInvite, &invite.InviteSent); err != nil {
				log.Printf("Unable to retrieve row from database result: %v\n", err)
			}
			invitations = append(invitations, invite)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Error retrieving collection invitations from database result: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(invitations)
		return

	} else if r.Method == "POST" {
		log.Printf("Invitations POST - Not implemented.")
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
		log.Printf("Invitations DELETE - Not implemented.")
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
