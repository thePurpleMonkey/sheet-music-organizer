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
	"github.com/lib/pq"
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
		var inviterID, invitationID int64
		var retracted bool
		if err := db.QueryRow("SELECT invitation_id, invitee_email, admin_invite, collection_id, inviter_id, retracted FROM invitations WHERE token = $1", token).Scan(&invitationID, &invite.Email, &invite.Administrator, &collectionID, &inviterID, &retracted); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Invitations GET - Attempted to accept invitation with invalid token: %v\n", token)
				SendError(w, `{"error": "Invitation not found."}`, http.StatusNotFound)
			} else {
				log.Printf("Invitations GET - Unable to get invitation from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			}
			return
		}

		// Check if invitation has been retracted
		if retracted {
			log.Printf("Invitations POST - User %d attempted to get a retracted invitation %d\n", session.Values["user_id"], invitationID)
			SendError(w, `{"error": "This invitation has been retracted and is no longer valid.", "code": "retracted"}`, http.StatusForbidden)
			return
		}

		// Check if the correct user is logged in
		if invite.Email != session.Values["email"] {
			log.Printf("Invitation GET - User %s logged in to accept invitation for %s.\n", session.Values["email"], invite.Email)
			SendError(w, `{"error": "You cannot accept this invitation. Please log out and try again.", "code": "wrong_user"}`, http.StatusForbidden)
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
		var collectionID, invitationID int64
		var adminInvite, retracted bool
		if err := db.QueryRow("SELECT admin_invite, collection_id, retracted, invitation_id FROM invitations WHERE token = $1", accept.Token).Scan(&adminInvite, &collectionID, &retracted, &invitationID); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Invitations POST - Attempted to accept invitation with invalid token: %v\n", accept.Token)
				SendError(w, `{"error": "Invitation not found."}`, http.StatusNotFound)
			} else {
				log.Printf("Invitations POST - Unable to accept invitation from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			}
			return
		}

		// Check if invitation has been retracted
		if retracted {
			log.Printf("Invitations POST - User %d accepted a retracted invitation %d\n", session.Values["user_id"], invitationID)
			SendError(w, `{"error": "This invitation has been retracted and is no longer valid."}`, http.StatusForbidden)
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
		// Get all invitations from this user for this collection
		rows, err := db.Query("SELECT invitation_id, invitee_email, admin_invite, invite_sent FROM invitations WHERE inviter_id = $1 AND collection_id = $2 AND retracted = false AND invite_sent > CURRENT_TIMESTAMP - INTERVAL '7 days'", session.Values["user_id"], collectionID)
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
			SendError(w, PERMISSION_ERROR_MESSAGE, http.StatusForbidden)
			return
		}

		// Check if user is an administrator on this collection
		var admin bool
		if err := db.QueryRow("SELECT admin FROM collection_members WHERE collection_id = $1 AND user_id = $2", collectionID, session.Values["user_id"]).Scan(&admin); err != nil {
			log.Printf("Invitations POST - Unable to get collection member from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		if !admin {
			log.Printf("Invitations POST - Non-admin user %d attempted to send invite to %s\n", session.Values["user_id"], invite.InviteeEmail)
			SendError(w, `{"error": "You are not authorized to perfrom this action for this collection."}`, http.StatusForbidden)
			return
		}

		// Check for existing invitations
		var invitationID int64
		var sent time.Time
		if err := db.QueryRow("SELECT invitation_id, invite_sent FROM invitations WHERE collection_id = $1 AND inviter_id = $2 AND invitee_email = $3", collectionID, session.Values["user_id"], invite.InviteeEmail).Scan(&invitationID, &sent); err != nil {
			// There was a database error executing the SQL statement
			if err != sql.ErrNoRows {
				log.Printf("Invitations POST - Unable to look up invitation for user")
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
				return
			}

			// The error returned was sql.ErrNoRows, so there is no existing invitation.
			// This new invitation can be safely committed to the database
		} else {
			// Invitation exists.
			log.Printf("Invitations POST - Existing invitation %d sent at %v.\n", invitationID, sent)
			if sent.Before(time.Now().AddDate(0, 0, -7)) {
				// Delete existing expired invitation from database
				if _, err = db.Exec("DELETE FROM invitations WHERE inviter_id = $1 AND invitee_email = $2, collection_id = $3", session.Values["user_id"], invite.InviteeEmail, collectionID); err != nil {
					log.Printf("Invitations POST - Unable to delete expired invite from database: %v\n", err)
					SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
					return
				}
				log.Printf("Invitations POST - Expired invitation deleted.")
			}

			// There was an invitation that has not expired yet.
			// Continue to the next step, which will fail because of the
			// UNIQUE constraint on the database table. That will
			// cause an appropriate "already invited" message to be
			// sent to the user.
		}

		// Add new invitation to database
		token := uniuri.NewLen(64)
		if _, err = db.Exec("INSERT INTO invitations (inviter_id, invitee_email, admin_invite, collection_id, token) VALUES ($1, $2, $3, $4, $5)", session.Values["user_id"], invite.InviteeEmail, invite.AdminInvite, collectionID, token); err != nil {
			if pgerr, ok := err.(*pq.Error); ok {
				if pgerr.Code == "23505" {
					log.Printf("Invitations POST - User %d attempted to re-invite user %s\n", session.Values["user_id"], invite.InviteeEmail)
					SendError(w, `{"error": "There is already a pending invitation for this user."}`, http.StatusConflict)
				} else {
					log.Printf("Invitations POST - Unable to create invite in database: %v\n", err)
					SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
				}
			} else {
				log.Printf("Invitations POST - Unable to create invite in database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			}
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
		htmlTemplate := template.Must(template.New("invite_email.html").ParseFiles("email_templates/invite_email.html"))
		textTemplate := template.Must(template.New("invite_email.txt").ParseFiles("email_templates/invite_email.txt"))

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
		// Get invitation_id from URL
		invitationID, err := strconv.ParseInt(mux.Vars(r)["invitation_id"], 10, 64)
		if err != nil {
			log.Printf("Invitation DELETE - Unable to parse invitation id: %v\n", err)
			SendError(w, URL_ERROR_MESSAGE, http.StatusBadRequest)
			return
		}

		// Check if user actually sent this invitation
		var inviterID int64
		if err := db.QueryRow("SELECT inviter_id FROM invitations WHERE collection_id = $1 AND invitation_id = $2", collectionID, invitationID).Scan(&inviterID); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Invitation DELETE - User %d attempted to retract a non-existant invitation %d.\n", session.Values["user_id"], invitationID)
				SendError(w, `{"error": "Invitation not found."}`, http.StatusNotFound)
			} else {
				log.Printf("Invitation DELETE - Unable to get collection member from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			}
			return
		}
		if inviterID != session.Values["user_id"] {
			log.Printf("Invitation DELETE - User %d attempted to retract invitation %d owned by user %d.\n", session.Values["user_id"], invitationID, inviterID)
			SendError(w, PERMISSION_ERROR_MESSAGE, http.StatusForbidden)
			return
		}

		// Mark invitation as retracted
		if _, err = db.Exec("UPDATE invitations SET retracted = true WHERE invitation_id = $1", invitationID); err != nil {
			log.Printf("Invitation DELETE - Unable to retract invitation %d: %v\n", invitationID, err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}
