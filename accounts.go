package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
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
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
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
