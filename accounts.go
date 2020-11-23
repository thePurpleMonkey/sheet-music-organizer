package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/dchest/uniuri"
)

// AccountHandler handles getting and updating account information, as well as requesting account deletion
func AccountHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Account handler - Unable to get session: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		var user User

		// Retrieve user from database
		err := db.QueryRow("SELECT user_id, email, name, verified, restricted FROM users WHERE user_id = $1", session.Values["user_id"]).Scan(&user.UserID, &user.Email, &user.Name, &user.Verified, &user.Restricted)
		if err != nil {
			log.Printf("Account GET - Unable to get user from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send response
		log.Printf("Account GET - Retrieved user account %d\n", session.Values["user_id"])
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
		return
	} else if r.Method == "PUT" {
		var user User

		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Account PUT - Unable to parse request body: %v\n", err)
			body, _ := ioutil.ReadAll(r.Body)
			log.Printf("Body: %v\n", body)
			SendError(w, `{"error": "Unable to parse request."}`, http.StatusBadRequest)
			return
		}

		// Check if updating email address
		var unverify = false
		if user.Email != session.Values["email"] {
			unverify = true
		}

		// Calculate user's new verified status
		var verified = session.Values["verified"].(bool) && !unverify

		// Unverify user's session if necessary
		if unverify {
			log.Printf("Account PUT - Unverifying user account %d in session.\n", session.Values["user_id"])
			session.Values["verified"] = false
			if err = session.Save(r, w); err != nil {
				log.Printf("Account PUT - Unable to save session state: %v\n", err)
				SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
				return
			}
		}

		// Update user in database
		var result sql.Result
		if result, err = db.Exec("UPDATE users SET name = $1, email = $2, verified = $3 WHERE user_id = $4", user.Name, user.Email, verified, session.Values["user_id"]); err != nil {
			log.Printf("Account PUT - Unable to update user in database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Check if update did anything
		var rows int64
		rows, err = result.RowsAffected()
		if err != nil {
			log.Printf("Account PUT - Unable to get rows affected by update: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		if rows == 0 {
			log.Printf("Account PUT - User %d update did not affect any row in database", session.Values["user_id"])
			SendError(w, `{"error": "User not found."}`, http.StatusNotFound)
			return
		}

		log.Printf("Account PUT - Updated user %d\n", session.Values["user_id"])
		w.WriteHeader(http.StatusOK)
		return
	} else if r.Method == "DELETE" {
		// Start db transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Collection DELETE - Unable to start database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Get a list of all user's collections
		rows, err := tx.Query("SELECT collection_id FROM collection_members NATURAL JOIN collections WHERE user_id = $1", session.Values["user_id"])
		if err != nil {
			log.Printf("Account DELETE - Unable to retrieve collections from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Retrieve rows from database
		collections := make([]int64, 0)
		for rows.Next() {
			var collectionID int64
			if err := rows.Scan(&collectionID); err != nil {
				log.Printf("Unable to retrieve row from database result: %v\n", err)
			}
			collections = append(collections, collectionID)
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Printf("Error retrieving collections from database result: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Iterate through each collection
		for _, collectionID := range collections {
			// Check if user is the sole admin for this collection
			var remainingAdmins int64
			if err = tx.QueryRow("SELECT COUNT(*) FROM collection_members WHERE collection_id = $1 AND admin = true AND user_id != $2", collectionID, session.Values["user_id"]).Scan(&remainingAdmins); err != nil {
				log.Printf("Account DELETE - Unable to get the remaining admins from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
				return
			}

			if remainingAdmins == 0 {
				// Delete the collection
				if err = deleteCollection(collectionID, tx); err != nil {
					log.Printf("Account DELETE - Unable to delete collection %d for user %d.\n", collectionID, session.Values["user_id"])
					SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
					return
				}
			} else {
				// Leave the collection
				if _, err = tx.Exec("DELETE FROM collection_members WHERE user_id = $1 AND collection_id = $2", session.Values["user_id"], collectionID); err != nil {
					log.Printf("Account DELETE - User %d unable to leave collection %d: %v\n", session.Values["user_id"], collectionID, err)
					SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
					return
				}
			}

			log.Printf("Account DELETE - Deleted collection %d for user %d\n", collectionID, session.Values["user_id"])
		}

		// Delete pending invitations
		if _, err = tx.Exec("DELETE FROM invitations WHERE inviter_id = $1", session.Values["user_id"]); err != nil {
			log.Printf("Account DELETE - Unable to delete pending invitations for user %d.\n", session.Values["user_id"])
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Delete any password reset records
		if _, err = tx.Exec("DELETE FROM password_reset WHERE user_id = $1", session.Values["user_id"]); err != nil {
			log.Printf("Account DELETE - Unable to delete password reset record for user %d.\n", session.Values["user_id"])
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Delete any verification email records
		if _, err = tx.Exec("DELETE FROM verification_emails WHERE user_id = $1", session.Values["user_id"]); err != nil {
			log.Printf("Account DELETE - Unable to delete verification email record for user %d.\n", session.Values["user_id"])
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Finally, delete the user record
		if _, err = tx.Exec("DELETE FROM users WHERE user_id = $1", session.Values["user_id"]); err != nil {
			log.Printf("Account DELETE - Unable to delete user record for user %d.\n", session.Values["user_id"])
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Save changes
		if err := tx.Commit(); err != nil {
			log.Printf("Account DELETE - Unable to commit database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Revoke user's authentication
		session.Values["authenticated"] = false
		if err = session.Save(r, w); err != nil {
			log.Printf("Account DELETE - Unable to save session state: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		log.Printf("Account DELETE - Deleted user %d\n", session.Values["user_id"])
		w.WriteHeader(http.StatusOK)
		return
	}
}

// VerifyHandler handles verifying account and sending emails
func VerifyHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Verify handler - Unable to get session: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		var token string = r.URL.Query().Get("token")

		if token == "" {
			SendError(w, `{"error": "No token provided."}`, http.StatusBadRequest)
			return
		}

		// Get user from database
		var userID int64
		if err := db.QueryRow("SELECT user_id FROM verification_emails WHERE token = $1", token).Scan(&userID); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Verify GET - Attempted to verify account with invalid token: %v\n", token)
				SendError(w, `{"error": "There was a problem verifying your account. Please try again."}`, http.StatusNotFound)
			} else {
				log.Printf("Verify GET - Unable to get verification record from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			}
			return
		}

		// Check if the correct user is logged in
		if userID != session.Values["user_id"] {
			log.Printf("Verify GET - User %d logged in to verify account for %d.\n", session.Values["user_id"], userID)
			SendError(w, `{"error": "There was a problem verifying your account. Please try again."}`, http.StatusForbidden)
			return
		}

		// Start db transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Verify GET - Unable to begin database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}

		// Update the user in the database
		if _, err = tx.Exec("UPDATE users SET verified = true WHERE user_id = $1", userID); err != nil {
			log.Printf("Verify GET - Unable to update user record in database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Delete the invite
		if _, err = tx.Exec("DELETE FROM verification_emails WHERE token = $1", token); err != nil {
			log.Printf("Verify GET - Unable to delete verification email record from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Save changes
		if err = tx.Commit(); err != nil {
			log.Printf("Verify GET - Unable to commit database transaction: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}

		// Update session
		log.Printf("Verify GET - Verifying user %d's session.", session.Values["user_id"])
		session.Values["verified"] = true
		if err = session.Save(r, w); err != nil {
			log.Printf("Verify GET - Unable to save session state: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		log.Printf("Verify GET - User %d verified.\n", userID)
		w.WriteHeader(http.StatusOK)
		return
	} else if r.Method == "POST" {
		var user User
		var userID int64 = session.Values["user_id"].(int64)
		log.Printf("Verify POST - Sending verification email for user %d\n", userID)
		// Check for the user in the database
		if err := db.QueryRow("SELECT name, email, verified FROM users WHERE user_id = $1", userID).Scan(&user.Name, &user.Email, &user.Verified); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Verify POST - Verification email requested for non-existent user %v\n", userID)
				w.WriteHeader(http.StatusOK)
			} else {
				log.Printf("Verify POST - Unable to retrieve user from database: %v\n", err)
				SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			}
			return
		}

		// Create new verification email record in database
		token := uniuri.NewLen(64)
		if _, err := db.Exec("INSERT INTO verification_emails (user_id, token) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET user_id = $1, token = $2, expires = CURRENT_TIMESTAMP + interval '24 hours'", userID, token); err != nil {
			log.Printf("Verify POST - Unable to insert verification email record into database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Create email template
		htmlTemplate := template.Must(template.New("verification_email.html").ParseFiles("email_templates/verification_email.html"))
		textTemplate := template.Must(template.New("verification_email.txt").ParseFiles("email_templates/verification_email.txt"))

		var htmlBuffer, textBuffer bytes.Buffer
		url := "https://" + os.Getenv("HOST") + "/verify.html?token=" + token
		data := struct {
			Href string
			Name string
		}{url, user.Name}

		if err := htmlTemplate.Execute(&htmlBuffer, data); err != nil {
			log.Printf("Verify POST - Unable to execute html template: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		if err := textTemplate.Execute(&textBuffer, data); err != nil {
			log.Printf("Verify POST - Unable to execute text template: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send email
		if err := SendEmail(user.Name, user.Email, "Email Verification", htmlBuffer.String(), textBuffer.String()); err != nil {
			log.Printf("Verify POST - Failed to send verification email: %v\n", err)
			SendError(w, `{"error": "Unable to send verification email."}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
