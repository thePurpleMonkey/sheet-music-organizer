package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

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

	if checkPasswordHash(user.Password, hashedPassword) {

		// Set user as authenticated
		session.Values["authenticated"] = true
		session.Values["name"] = name
		if err := session.Save(r, w); err != nil {
			log.Fatal(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Incorrect email or password"}`))
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

	// Parse and decode the request body into a new `Credentials` instance
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
	session.Save(r, w)
}
