package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// HTMLHandler handles serving templated HTML pages
func HTMLHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	filename := vars["filename"]

	// Load template
	t, err := template.ParseGlob("html_templates/*.html")
	if err != nil {
		log.Printf("HTML Handler - Unable to parse file %s: %v\n", filename, err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	tmpl := t.Lookup(filename + ".html")
	if tmpl == nil {
		// The desired template was not found, so present a 404 error
		fmt.Printf("HTML Handler - Requested page %s.html does not exist.\n", filename)
		http.Redirect(w, r, "NotFound.html", http.StatusFound)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("HTML Handler - Unable to execute template %s.html: %v\n", filename, err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
	}
}
