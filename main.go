package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

func makeRouter() *mux.Router {
	r := mux.NewRouter()

	// Users
	r.HandleFunc("/user/login", login).Methods("POST")
	r.HandleFunc("/user/logout", logout)
	r.HandleFunc("/user/register", register).Methods("POST")

	// Collections
	r.HandleFunc("/collections", RequireAuthentication(CollectionsHandler)).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}", RequireAuthentication(CollectionHandler)).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}/songs", RequireAuthentication(SongsHandler)).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}/songs/{song_name}", RequireAuthentication(SongHandler)).Methods("GET", "POST")

	r.HandleFunc("/books/{title}/page/{page}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		title := vars["title"]
		page := vars["page"]

		fmt.Fprintf(w, "You've requested the book: %s on page %s\n", title, page)
	})

	// Static files
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("static"))))

	return r
}

func main() {
	// Check environment variables
	if os.Getenv("SESSION_KEY") == "" {
		panic("Session key not set!")
	} else {
		fmt.Println(os.Getenv("SESSION_KEY"))
	}

	// Initialize router
	r := makeRouter()

	// Connect to database
	var err error
	db, err = sql.Open("postgres", "user=smo dbname=smo password=smo-test sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Launch server
	fmt.Println("Running on port 8000")
	// log.Fatal(http.ListenAndServe(":8000", r))
	log.Fatal(http.ListenAndServe(":8000", handlers.RecoveryHandler()(r)))
}
