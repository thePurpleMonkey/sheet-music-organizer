package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// User is a struct that models the structure of a user, both in the request body, and in the DB
type User struct {
	Password string `json:"password", db:"password"`
	Email    string `json:"email", db:"email"`
	Name     string `json:"name", db:"name"`
}

// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func login(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
		return
	}

	// Parse and decode the request body into a new `User` instance
	user := &User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		// If there is something wrong with the request body, return a 400 status
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Pull user with email from
	var hashedPassword, name string
	if err := db.QueryRow("SELECT password, name FROM users WHERE email = $1", user.Email).Scan(&hashedPassword, &name); err != nil {
		if err == sql.ErrNoRows {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Incorrect email or password"}`))
		} else {
			log.Fatal(err)
		}
		return
	}

	if !checkPasswordHash(user.Password, hashedPassword) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Incorrect email or password"}`))
		return
	}

	// Get a list of all user's collections
	rows, err := db.Query("SELECT collection_id FROM collection_members WHERE user_email = $1", user.Email)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Error retrieving your collections from database."}`))
		log.Fatalf("Login - Unable to retrieve collection ids from database: %v\n", err)
		return
	}
	defer rows.Close()

	// Retrieve rows from database
	IDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			log.Printf("Login - Error scanning rows: %v\n", err)
		}
		IDs = append(IDs, id)
	}

	// Check for errors from iterating over rows.
	if err := rows.Err(); err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Error retrieving collections from database."}`))
		log.Fatalf("Login - Unable to read collection IDs from database: %v\n", err)
		return
	}

	// Set user as authenticated
	session.Values["authenticated"] = true
	session.Values["name"] = name
	session.Values["email"] = user.Email
	session.Values["ids"] = IDs
	if err := session.Save(r, w); err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(r, w)

	w.WriteHeader(http.StatusOK)
}

func register(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")

	// Parse and decode the request body into a new `User` instance
	user := &User{}
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		// If there is something wrong with the request body, return a 400 status
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hashedPass, err := hashPassword(user.Password)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Create user in database
	if _, err = db.Exec("INSERT INTO users VALUES ($1, $2, $3)", user.Email, hashedPass, user.Name); err != nil {
		if err.(*pq.Error).Code == "23505" {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Email already registered"}`))
		} else {
			log.Fatal(err)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)

	// Set user as authenticated
	session.Values["authenticated"] = true
	session.Values["name"] = user.Name
	session.Values["email"] = user.Email
	session.Save(r, w)
}

// RequireAuthentication is a middleware that checks if the user is authenticated,
// and returns a 403 Forbidden error if not.
func RequireAuthentication(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Fatal(err)
			return
		}

		// Check if user is authenticated
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		f(w, r)
	}
}

// VerifyCollectionID is a middleware that checks if the user is authorized
// to access a collection, and returns a 403 Forbidden error if not.
func VerifyCollectionID(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Fatal(err)
			return
		}

		acceptableIDs := session.Values["ids"].([]int)

		// Get URL parameter
		collectionID, err := strconv.Atoi(mux.Vars(r)["collection_id"])
		if err != nil {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Unable to parse collection id."}`))
			log.Printf("Collection ID middleware - Unable to get collection id: %v\n", err)
			return
		}

		for _, id := range acceptableIDs {
			if collectionID == id {
				f(w, r)
				return
			}
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}
