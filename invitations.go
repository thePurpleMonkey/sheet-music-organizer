package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/dchest/uniuri"
	"github.com/gorilla/mux"
)

// Invite is a struct that models the invitations to a collection
type Invite struct {
	InvitationID int64      `json:"invitation_id" db:"invitation_id"`
	InviteeEmail string     `json:"invitee_email" db:"invitee_email"`
	InviteeName  string     `json:"invitee_name"`
	AdminInvite  bool       `json:"admin_invite" db:"admin_invite"`
	InviteSent   *time.Time `json:"invite_sent" db:"invite_sent"`
	Message      string     `json:"message"`
}

// AcceptInvite is a struct that is sent to a user confirming an invitation
type AcceptInvite struct {
	Email          string `json:"email"`
	CollectionName string `json:"collection_name"`
	InviterName    string `json:"inviter_name"`
	InviterEmail   string `json:"inviter_email"`
	Administrator  bool   `json:"administrator"`
}

// InvitationsHandler handles accepting invitations
func InvitationsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Invitations handler - Unable to get session: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		var token string = r.URL.Query().Get("token")

		if token == "" {
			SendError(w, `{"error": "No token provided."}`, http.StatusBadRequest)
			return
		}

		// Get invitation from database
		var invite AcceptInvite
		var collectionID string
		var inviterID int64
		if err := db.QueryRow("SELECT invitee_email, admin_invite, collection_id, inviter_id FROM invitations WHERE token = $1", token).Scan(&invite.Email, &invite.Administrator, &collectionID, &inviterID); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Invitations GET - Attempted to accept invitation with invalid token: %v\n", token)
				SendError(w, `{"error": "Invitation not found."}`, http.StatusNotFound)
			} else {
				log.Printf("Invitations GET - Unable to get invitation from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			}
			return
		}

		// Check if the correct user is logged in
		if invite.Email != session.Values["email"] {
			log.Printf("Invitation GET - User %s logged in to accept invitation for %s.\n", session.Values["email"], invite.Email)
			SendError(w, `{"error": "You cannot accept this invitation. Please log out and try again."}`, http.StatusForbidden)
			return
		}

		// Get collection from database
		if err := db.QueryRow("SELECT name FROM collections WHERE collection_id = $1", collectionID).Scan(&invite.CollectionName); err != nil {
			log.Printf("Invitations GET - Unable to get invitation's collection name from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Get inviter from database
		if err := db.QueryRow("SELECT name, email FROM users WHERE user_id = $1", inviterID).Scan(&invite.InviterName, &invite.InviterEmail); err != nil {
			log.Printf("Invitations GET - Unable to get invitation's inviter from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(invite)
		return
	} else if r.Method == "POST" {
		var accept struct {
			Token string `json:"token"`
		}
		err := json.NewDecoder(r.Body).Decode(&accept)
		if err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Invitations POST - Unable to parse request body: %v\n", err)
			body, _ := ioutil.ReadAll(r.Body)
			log.Printf("Body: %s\n", body)
			SendError(w, `{"error": "Unable to parse request body."}`, http.StatusBadRequest)
			return
		}

		// Get invitation from database
		var collectionID int64
		var adminInvite bool
		if err := db.QueryRow("SELECT admin_invite, collection_id FROM invitations WHERE token = $1", accept.Token).Scan(&adminInvite, &collectionID); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Invitations POST - Attempted to accept invitation with invalid token: %v\n", accept.Token)
				SendError(w, `{"error": "Invitation not found."}`, http.StatusNotFound)
			} else {
				log.Printf("Invitations POST - Unable to accept invitation from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			}
			return
		}

		// Start db transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Invitations POST - Unable to begin database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}

		// Add the user to this collection
		if _, err = tx.Exec("INSERT INTO collection_members VALUES ($1, $2, $3)", session.Values["user_id"], collectionID, adminInvite); err != nil {
			log.Printf("Invitations POST - Unable to add  user to collection: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Delete the invite
		if _, err = tx.Exec("DELETE FROM invitations WHERE token = $1", accept.Token); err != nil {
			log.Printf("Invitations POST - Remove invite from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Save changes
		if err = tx.Commit(); err != nil {
			log.Printf("Invitations POST - Unable to commit database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}

		// Make the new collection available for this user's session
		session.Values["ids"] = append(session.Values["ids"].([]int64), collectionID)
		if err := session.Save(r, w); err != nil {
			log.Printf("Invitations POST - Unable to save session state: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
	}
}

// CollectionInvitationsHandler handles inviting new members, managing invitations, and revoking invitations to a collection.
func CollectionInvitationsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		log.Printf("Invitations handler - Unable to get session: %v\n", err)
		return
	}

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
		var invite Invite
		err := json.NewDecoder(r.Body).Decode(&invite)
		if err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Invitations POST - Unable to parse request body: %v\n", err)
			body, _ := ioutil.ReadAll(r.Body)
			log.Printf("Body: %s\n", body)
			SendError(w, `{"error": "Unable to parse request body."}`, http.StatusBadRequest)
			return
		}

		// Check if user is allowed to send invitations
		if session.Values["restricted"].(bool) {
			log.Printf("Invitations POST - Restricted user '%s' attempted to send an invitation to '%s'\n", session.Values["email"], invite.InviteeEmail)
			SendError(w, `{"error": "That action is not permitted."}`, http.StatusForbidden)
			return
		}

		// Update collection in database
		token := uniuri.NewLen(64)
		if _, err = db.Exec("INSERT INTO invitations (inviter_id, invitee_email, admin_invite, collection_id, token) VALUES ($1, $2, $3, $4, $5)", session.Values["user_id"], invite.InviteeEmail, invite.AdminInvite, collectionID, token); err != nil {
			log.Printf("Invitations POST - Unable to create invite in database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Get collection name from database
		var collectionName string
		if err := db.QueryRow("SELECT name FROM collections WHERE collection_id = $1", collectionID).Scan(&collectionName); err != nil {
			log.Printf("Invitations POST - Unable to get collection name from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Create email template
		htmlTemplate := template.Must(template.New("invite_email.html").ParseFiles("templates/invite_email.html"))
		textTemplate := template.Must(template.New("invite_email.txt").ParseFiles("templates/invite_email.txt"))

		var htmlBuffer, textBuffer bytes.Buffer
		url := "https://" + os.Getenv("HOST") + "/accept_invite.html?token=" + token
		data := struct {
			Href           string
			InviteeName    string
			InviterName    string
			InviterEmail   string
			CollectionName string
			Message        string
		}{
			url,
			invite.InviteeName,
			session.Values["name"].(string),
			session.Values["email"].(string),
			collectionName,
			invite.Message,
		}

		if err := htmlTemplate.Execute(&htmlBuffer, data); err != nil {
			log.Printf("Invitation - Unable to execute html template: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		if err := textTemplate.Execute(&textBuffer, data); err != nil {
			log.Printf("Invitation - Unable to execute text template: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send email
		if err := SendEmail(invite.InviteeName, invite.InviteeEmail, "Sheet Music Organizer Invitation", htmlBuffer.String(), textBuffer.String()); err != nil {
			log.Printf("Invitation - Failed to invitation email: %v\n", err)
			SendError(w, `{"error": "Unable to send invitation email."}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		return

	} else if r.Method == "DELETE" {
		log.Printf("Invitations DELETE - Not implemented.")
		SendError(w, "Not Found", http.StatusNotFound)
		return

		// // Start db transaction
		// tx, err := db.Begin()
		// if err != nil {
		// 	log.Printf("Collection DELETE - Unable to start database transaction: %v\n", err)
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// }

		// // Remove songs from collection
		// if _, err = tx.Exec("DELETE FROM songs WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	log.Printf("Collection DELETE - Unable to delete songs from collection: %v\n", err)
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// // Remove tags from collection
		// if _, err = tx.Exec("DELETE FROM tags WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	log.Printf("Collection DELETE - Unable to delete tags from collection: %v\n", err)
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// // Remove users from collection
		// if _, err = tx.Exec("DELETE FROM collection_members WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	log.Printf("Collection DELETE - Unable to delete collection members: %v\n", err)
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// // Delete collection
		// if _, err = tx.Exec("DELETE FROM collections WHERE collection_id = $1", collection.CollectionID); err != nil {
		// 	log.Printf("Collection DELETE - Unable to delete collection from database: %v\n", err)
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// // Save changes
		// if err = tx.Commit(); err != nil {
		// 	log.Printf("Collection DELETE - Unable to commit database transaction: %v\n", err)
		// 	SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		// 	return
		// }

		// w.WriteHeader(http.StatusOK)
		// return
	}
}
