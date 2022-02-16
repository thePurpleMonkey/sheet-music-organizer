package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

func preventDirectoryListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func makeRouter() *mux.Router {
	r := mux.NewRouter()

	// Users
	r.HandleFunc("/user/login", login).Methods("POST")
	r.HandleFunc("/user/logout", logout)
	r.HandleFunc("/user/register", register).Methods("POST")
	r.HandleFunc("/user/password/forgot", requestPasswordResetEmail).Methods("POST")
	r.HandleFunc("/user/password/reset", resetPassword)
	r.HandleFunc("/user/account", RequireAuthentication(AccountHandler)).Methods("GET", "PUT", "DELETE")
	r.HandleFunc("/user/verify", RequireAuthentication(VerifyHandler)).Methods("GET", "POST")
	r.HandleFunc("/invitations", RequireAuthentication(InvitationsHandler)).Methods("GET", "POST")
	r.HandleFunc("/user/invitations", RequireAuthentication(UserInvitationsHandler)).Methods("GET", "POST")

	// Collections
	r.HandleFunc("/collections", RequireAuthentication(CollectionsHandler)).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}", VerifyCollectionID(RequireAuthentication(CollectionHandler))).Methods("GET", "PUT", "DELETE")
	r.HandleFunc("/collections/{collection_id}/members", VerifyCollectionID(RequireAuthentication(MembersHandler))).Methods("GET")
	r.HandleFunc("/collections/{collection_id}/members/{user_id}", VerifyCollectionID(RequireAuthentication(MemberHandler))).Methods("PUT", "DELETE")
	r.HandleFunc("/collections/{collection_id}/invitations", VerifyCollectionID(RequireAuthentication(CollectionInvitationsHandler))).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}/invitations/{invitation_id}", VerifyCollectionID(RequireAuthentication(CollectionInvitationsHandler))).Methods("DELETE")
	r.HandleFunc("/collections/{collection_id}/search", VerifyCollectionID(RequireAuthentication(SearchHandler))).Methods("GET", "POST")

	// Songs
	r.HandleFunc("/collections/{collection_id}/songs", VerifyCollectionID(RequireAuthentication(SongsHandler))).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}/songs/{song_id}", VerifyCollectionID(RequireAuthentication(SongHandler))).Methods("GET", "PUT", "DELETE")
	r.HandleFunc("/collections/{collection_id}/songs/{song_id}/tags", VerifyCollectionID(RequireAuthentication(SongTagsHandler))).Methods("GET", "POST", "DELETE")

	// Tags
	r.HandleFunc("/collections/{collection_id}/tags", VerifyCollectionID(RequireAuthentication(TagsHandler))).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}/tags/{tag_id}", VerifyCollectionID(RequireAuthentication(TagHandler))).Methods("GET", "PUT", "DELETE")
	r.HandleFunc("/collections/{collection_id}/tags/{tag_id}/songs", VerifyCollectionID(RequireAuthentication(TagSongsHandler))).Methods("GET")

	// Setlists
	r.HandleFunc("/collections/{collection_id}/setlists", VerifyCollectionID(RequireAuthentication(SetlistsHandler))).Methods("GET", "POST")
	r.HandleFunc("/collections/{collection_id}/setlists/{setlist_id}", VerifySetlistID(VerifyCollectionID(RequireAuthentication(SetlistHandler)))).Methods("GET", "PUT", "DELETE")
	r.HandleFunc("/collections/{collection_id}/setlists/{setlist_id}/visibility", VerifySetlistID(VerifyCollectionID(RequireAuthentication(SetlistVisibilityHandler)))).Methods("PUT")
	r.HandleFunc("/collections/{collection_id}/setlists/{setlist_id}/songs", VerifySetlistID(VerifyCollectionID(RequireAuthentication(SetlistSongsHandler)))).Methods("GET", "POST", "PUT")
	r.HandleFunc("/collections/{collection_id}/setlists/{setlist_id}/songs/{song_id}", VerifySetlistID(VerifyCollectionID(RequireAuthentication(SetlistSongHandler)))).Methods("DELETE")

	// Public setlist
	r.HandleFunc("/setlists/{share_code}", PublicSetlistHandler).Methods("GET")
	r.HandleFunc("/setlists/{share_code}/songs", PublicSetlistSongsHandler).Methods("GET")

	// Contact Us
	r.HandleFunc("/contact", ContactHandler).Methods("POST")

	// r.HandleFunc("/books/{title}/page/{page}", func(w http.ResponseWriter, r *http.Request) {
	// 	vars := mux.Vars(r)
	// 	title := vars["title"]
	// 	page := vars["page"]

	// 	fmt.Fprintf(w, "You've requested the book: %s on page %s\n", title, page)
	// })

	// Static files
	r.HandleFunc("/{language}/{filename}.html", HTMLHandler)
	r.HandleFunc("/{filename}.html", HTMLHandler)
	r.HandleFunc("/", HTMLHandler)
	r.PathPrefix("/").Handler(http.StripPrefix("/", preventDirectoryListing(http.FileServer(http.Dir("static")))))

	return r
}

func main() {
	log.Println()
	log.Println("==============================")
	log.Println("Server booted")

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

	if os.Getenv("ADMIN_EMAIL") == "" {
		panic("Administrator email address not set!")
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
	log.Printf("Running on port %s\n", port)
	log.Fatal(http.ListenAndServeTLS(":"+port, os.Getenv("CERT_FILE"), os.Getenv("KEY_FILE"), handlers.RecoveryHandler()(r)))
}
