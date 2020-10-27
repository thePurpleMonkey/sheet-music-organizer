package main

import (
	"log"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Generic error message for database errors
var DATABASE_ERROR_MESSAGE = `{"error": "Error communicating with database."}`

// Generic error message for server errors
var SERVER_ERROR_MESSAGE = `{"error": "There was an error attempting to complete this operation. Please try again later."}`

// Generic error message for parsing URLs
var URL_ERROR_MESSAGE = `{"error": "Unable to parse URL."}`

// SendEmail ...
func SendEmail(name string, address string, subject string, htmlContent string, plainTextContent string) error {
	from := mail.NewEmail("Sheet Music Organizer", "sheetmusicorganizer@michaelhumphrey.dev")
	to := mail.NewEmail(name, address)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	log.Printf("Email sent! Status code: %v\n", response.StatusCode)
	log.Printf("Email body: %v\n", response.Body)
	if err != nil {
		return err
	}

	return nil
}
