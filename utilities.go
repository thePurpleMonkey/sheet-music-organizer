package main

import (
	"log"
	"net/http"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// DATABASE_ERROR_MESSAGE is a generic error message for database errors
const DATABASE_ERROR_MESSAGE string = `{"error": "Error communicating with database."}`

// SERVER_ERROR_MESSAGE is a generic error message for server errors
const SERVER_ERROR_MESSAGE string = `{"error": "There was an error attempting to complete this operation. Please try again later."}`

// URL_ERROR_MESSAGE is a generic error message for parsing URLs
const URL_ERROR_MESSAGE string = `{"error": "Unable to parse URL."}`

// REQUEST_ERROR_MESSAGE is a generic error message for parsing request bodies
const REQUEST_ERROR_MESSAGE string = `{"error": "Unable to parse request."}`

// PERMISSION_ERROR_MESSAGE is a generic error message for attempting an action that you do not have permission for.
const PERMISSION_ERROR_MESSAGE string = `{"error": "That action is not permitted."}`

// SendEmail ...
func SendEmail(name string, address string, subject string, htmlContent string, plainTextContent string) error {
	from := mail.NewEmail("Sheet Music Organizer", "support@sheetmusicorganizer.com")
	to := mail.NewEmail(name, address)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	log.Printf("Email sent! Status code: %v\n", response.StatusCode)
	log.Printf("Email response body: %v\n", response.Body)
	return err
}

// SendError ...
func SendError(w http.ResponseWriter, message string, httpCode int) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	w.Write([]byte(message))
}
