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
	if filename == "" {
		filename = "index"
	}

	language := vars["language"]
	if language == "" {
		language = "en"
	}

	// Load template
	template_path := fmt.Sprintf("html_templates/%s/*.html", language)
	t, err := template.ParseGlob(template_path)
	if err != nil {
		log.Printf("HTML Handler - Unable to parse file %s: %v\n", filename, err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	log.Printf("HTML Handler - Serving %s.html (%s)\n", filename, language)
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
