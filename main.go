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
	r.HandleFunc("/user/password/forgot", requestPasswordResetEmail).Methods("POST")
	r.HandleFunc("/user/password/reset", resetPassword)

	// Collections
	r.HandleFunc("/collections", RequireAuthentication(CollectionsHandler)).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}", VerifyCollectionID(RequireAuthentication(CollectionHandler))).Methods("GET", "PUT", "DELETE")

	// Songs
	r.HandleFunc("/collections/{collection_id}/songs", VerifyCollectionID(RequireAuthentication(SongsHandler))).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}/songs/{song_id}", VerifyCollectionID(RequireAuthentication(SongHandler))).Methods("GET", "PUT", "DELETE")
	r.HandleFunc("/collections/{collection_id}/songs/{song_id}/tags", VerifyCollectionID(RequireAuthentication(SongTagsHandler))).Methods("GET", "POST", "DELETE")

	// Tags
	r.HandleFunc("/collections/{collection_id}/tags", VerifyCollectionID(RequireAuthentication(TagsHandler))).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}/tags/{tag_id}", VerifyCollectionID(RequireAuthentication(TagHandler))).Methods("GET", "PUT", "DELETE")
	r.HandleFunc("/collections/{collection_id}/tags/{tag_id}/songs", VerifyCollectionID(RequireAuthentication(TagSongsHandler))).Methods("GET")

	// r.HandleFunc("/books/{title}/page/{page}", func(w http.ResponseWriter, r *http.Request) {
	// 	vars := mux.Vars(r)
	// 	title := vars["title"]
	// 	page := vars["page"]

	// 	fmt.Fprintf(w, "You've requested the book: %s on page %s\n", title, page)
	// })

	// Static files
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("static"))))

	return r
}

func main() {
	// Check environment variables
	if os.Getenv("SESSION_KEY") == "" {
		panic("Session key environment variable not set!")
	}

	if os.Getenv("DB_USERNAME") == "" {
		panic("Database username not set!")
	}

	if os.Getenv("DB_PASSWORD") == "" {
		panic("Database password not set!")
	}

	if os.Getenv("CERT_FILE") == "" {
		panic("Certificate path not set!")
	}

	if os.Getenv("KEY_FILE") == "" {
		panic("Key path not set!")
	}

	var port string
	if port = os.Getenv("PORT"); port == "" {
		port = "8000"
	}

	// Initialize router
	r := makeRouter()

	// Connect to database
	var err error
	db, err = sql.Open("postgres", "user="+os.Getenv("DB_USERNAME")+" dbname=smo password="+os.Getenv("DB_PASSWORD")+" sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Launch server
	fmt.Println("Running on port " + port)
	// log.Fatal(http.ListenAndServe(":8000", handlers.RecoveryHandler()(r)))
	log.Fatal(http.ListenAndServeTLS(":"+port, os.Getenv("CERT_FILE"), os.Getenv("KEY_FILE"), handlers.RecoveryHandler()(r)))
}
