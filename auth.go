package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/dchest/uniuri"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// User is a struct that models the structure of a user, both in the request body, and in the DB
type User struct {
	Password string `json:"password" db:"password"`
	Email    string `json:"email" db:"email"`
	Name     string `json:"name" db:"name"`
}

// PasswordResetRequest is a data structure to model incoming parameters of a password reset POST request
type PasswordResetRequest struct {
	Email    string `json:"email"`
	Token    string `json:"token"`
	Password string `json:"password"`
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

func requestPasswordResetEmail(w http.ResponseWriter, r *http.Request) {
	// Parse and decode the request body into a new `User` instance
	user := &User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		// If there is something wrong with the request body, return a 400 status
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Malformed request"}`))
		return
	}

	if len(user.Email) == 0 {
		log.Println("Email not provided in reset email request")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Email not provided"}`))
		return
	}

	// Check for the user in the database
	if err := db.QueryRow("SELECT email FROM users WHERE email = $1", user.Email).Scan(&user.Name); err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Password reset for non-existent user %s\n", user.Email)
			w.WriteHeader(http.StatusOK)
		} else {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error retrieving email from database."}`))
			log.Fatalf("Password Reset - Unable to retrieve  email from database: %v\n", err)
		}
		return
	}

	// Create password reset record in database
	token := uniuri.NewLen(64)
	if _, err := db.Exec("INSERT INTO password_reset VALUES ($1, $2, $3) ON CONFLICT (email) DO UPDATE SET email = $1, token = $2, expires = $3", user.Email, token, time.Now().Add(time.Hour)); err != nil {
		log.Fatal(err)
		return
	}

	// Create email template
	htmlTemplate := template.Must(template.New("password_reset_email.html").ParseFiles("templates/password_reset_email.html"))
	textTemplate := template.Must(template.New("password_reset_email.txt").ParseFiles("templates/password_reset_email.txt"))

	var htmlBuffer, textBuffer bytes.Buffer
	url := "https://" + os.Getenv("HOST") + "/reset_password.html?token=" + token + "&email=" + user.Email
	data := struct{ Href string }{url}

	htmlTemplate.Execute(&htmlBuffer, data)
	textTemplate.Execute(&textBuffer, data)

	// Send email
	if err := SendEmail(user.Name, user.Email, "Password Reset Email", htmlBuffer.String(), textBuffer.String()); err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Unable to send password reset email."}`))
		log.Fatalf("Password Reset - Failed to send email: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func resetPassword(w http.ResponseWriter, r *http.Request) {
	// Parse and decode the request body into a new `PasswordResetRequest` instance
	req := &PasswordResetRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		// If there is something wrong with the request body, return a 400 status
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Malformed request"}`))
		return
	}

	if len(req.Email) == 0 {
		log.Println("Email not provided in reset email request")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Email not provided"}`))
		return
	}

	if len(req.Token) == 0 {
		log.Println("Token not provided in reset email request")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Token not provided"}`))
		return
	}

	// Retrieve password reset request from database
	var expires time.Time
	var name string
	if err := db.QueryRow("SELECT expires, name FROM password_reset JOIN users ON users.email = password_reset.email WHERE password_reset.email = $1 AND token = $2", req.Email, req.Token).Scan(&expires, &name); err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Password reset not found for user %s\n", req.Email)
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error retrieving password reset request from database."}`))
			log.Fatalf("Password Reset - Unable to retrieve password reset request from database: %v\n", err)
		}
		return
	}

	if expires.Before(time.Now()) {
		// Password reset request expired
		log.Printf("User %v attempt to use expired password reset, which expired on %v\n", req.Email, expires)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "That password reset request has expired. Please request a new password reset email."}`))
		return
	}

	// User has valid password reset token. Let's reset the password!
	hashedPass, err := hashPassword(req.Password)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Update user in database
	if _, err = db.Exec("UPDATE users SET password = $1", hashedPass); err != nil {
		log.Printf("Reset Password - Unable to update user %v password! %v\n", req.Email, err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(DATABASE_ERROR_MESSAGE)
		return
	}

	// Delete password request from database
	if _, err := db.Exec("DELETE FROM password_reset WHERE email = $1", req.Email); err != nil {
		log.Printf("Unable to clear expired credentials from database: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(DATABASE_ERROR_MESSAGE)
	}

	// Set user as authenticated
	var session *sessions.Session
	if session, err = store.Get(r, "session"); err != nil {
		log.Printf("Unable to get session variables: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(DATABASE_ERROR_MESSAGE)
		return
	}
	session.Values["authenticated"] = true
	session.Values["name"] = name
	session.Values["email"] = req.Email
	session.Save(r, w)

	w.WriteHeader(http.StatusOK)
}

// RequireAuthentication is a middleware that checks if the user is authenticated,
// and returns a 403 Forbidden error if not.
func RequireAuthentication(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session")
		if err != nil {
			log.Printf("Require Authentication - Unable to get session: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if user is authenticated
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			log.Println("Attempt to access restricted page denied")
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

		acceptableIDs := session.Values["ids"].([]int64)

		// Get URL parameter
		collectionID, err := strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
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

		log.Printf("%v | %v not found in authorized collection IDs: %v", session.Values["email"], collectionID, acceptableIDs)
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}
