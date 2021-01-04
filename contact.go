package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"text/template"
)

// Message is a struct that models the structure of a message from the Contact Us form
type Message struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

// MessageTemplate is a struct for passing into the contact us template email
type MessageTemplate struct {
	Name    string
	Email   string
	Message string

	UserAuthenticated bool
	UserName          string
	UserEmail         string
	UserID            int64
	UserVerified      bool
	UserRestricted    bool
}

// ContactHandler handles sending an email from the Contact Us form
func ContactHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Contact handler - Unable to get session store: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	if r.Method == "POST" {
		// Parse and decode the request body into a new `Tag` instance
		message := &Message{}
		if err := json.NewDecoder(r.Body).Decode(message); err != nil {
			// If there is something wrong with the request body, return a 400 status
			log.Printf("Contact POST - Unable to decode request body: %v\n", err)
			SendError(w, REQUEST_ERROR_MESSAGE, http.StatusBadRequest)
			return
		}

		// Input validation
		if len(message.Email) == 0 {
			log.Println("Contact POST - Tag name not provided.")
			SendError(w, `{"error": "No name supplied."}`, http.StatusBadRequest)
			return
		}

		// Create email template
		htmlTemplate := template.Must(template.New("contact.html").ParseFiles("email_templates/contact.html"))
		textTemplate := template.Must(template.New("contact.txt").ParseFiles("email_templates/contact.txt"))

		var htmlBuffer, textBuffer bytes.Buffer

		var data MessageTemplate

		if _, authenticated := session.Values["authenticated"]; authenticated {
			data = MessageTemplate{
				message.Name,
				message.Email,
				message.Message,

				session.Values["authenticated"].(bool),
				session.Values["name"].(string),
				session.Values["email"].(string),
				session.Values["user_id"].(int64),
				session.Values["verified"].(bool),
				session.Values["restricted"].(bool),
			}
		} else {
			data = MessageTemplate{
				message.Name,
				message.Email,
				message.Message,

				false,
				"",
				"",
				0,
				false,
				false,
			}
		}
		if err := htmlTemplate.Execute(&htmlBuffer, data); err != nil {
			log.Printf("Contact POST - Unable to execute html template: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}
		if err := textTemplate.Execute(&textBuffer, data); err != nil {
			log.Printf("Contact POST - Unable to execute text template: %v\n", err)
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Send email
		if err := SendEmail("Sheet Music Organizer Site Administrator", os.Getenv("ADMIN_EMAIL"), "Sheet Music Organizer - Contact Us form", htmlBuffer.String(), textBuffer.String()); err != nil {
			log.Printf("Contact POST - Failed to send contact email: %v\n", err)
			SendError(w, `{"error": "Unable to send message."}`, http.StatusInternalServerError)
			return
		}

		// Respond with success
		w.WriteHeader(http.StatusOK)
	}
}
