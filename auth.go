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
	UserID   string `json:"user_id" db:"user_id"`
	Password string `json:"password" db:"password"`
	Email    string `json:"email" db:"email"`
	Name     string `json:"name" db:"name"`
}

// PasswordResetRequest is a data structure to model incoming parameters of a password reset POST request
type PasswordResetRequest struct {
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
		log.Printf("Login - Unable to retrieve session store: %v\n", err)
		SendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse and decode the request body into a new `User` instance
	user := &User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		// If there is something wrong with the request body, return a 400 status
		log.Printf("Login - Error decoding request body: %v\n", err)
		log.Printf("Body: %v\n", r.Body)
		SendError(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Pull user with email from
	var hashedPassword, name, userID string
	if err := db.QueryRow("SELECT password, name, user_id FROM users WHERE email = $1", user.Email).Scan(&hashedPassword, &name, &userID); err != nil {
		if err == sql.ErrNoRows {
			w.Header().Add("Content-Type", "application/json")
			SendError(w, `{"error": "Incorrect email or password"}`, http.StatusUnauthorized)
		} else {
			log.Printf("Login - Unable to retrieve username and password from database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}
		return
	}

	if !checkPasswordHash(user.Password, hashedPassword) {
		w.Header().Add("Content-Type", "application/json")
		SendError(w, `{"error": "Incorrect email or password"}`, http.StatusUnauthorized)
		return
	}

	// Get a list of all user's collections
	rows, err := db.Query("SELECT collection_id FROM collection_members WHERE user_id = $1", userID)
	if err != nil {
		log.Printf("Login - Unable to retrieve collection ids from database: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Retrieve rows from database
	IDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			log.Printf("Login - Error parsing collection IDs from database result: %v\n", err)
			continue
		}
		IDs = append(IDs, id)
	}

	// Check for errors from iterating over rows.
	if err := rows.Err(); err != nil {
		log.Printf("Login - Unable to read collection IDs from database: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Set user as authenticated
	session.Values["authenticated"] = true
	session.Values["name"] = name
	session.Values["email"] = user.Email
	session.Values["user_id"] = userID
	session.Values["ids"] = IDs
	if err := session.Save(r, w); err != nil {
		log.Printf("Login - Unable to save session state: %v\n", err)
		SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
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
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Register - Unable to retrieve session store: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Parse and decode the request body into a new `User` instance
	user := &User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		// If there is something wrong with the request body, return a 400 status
		log.Printf("Register - Unable to decode request body: %v", err)
		log.Printf("Body: %v\n", r.Body)
		SendError(w, `{"error": "Unable to decode request body."}`, http.StatusBadRequest)
		return
	}

	hashedPass, err := hashPassword(user.Password)
	if err != nil {
		log.Printf("Register - Unable to hash password: %v\n", err)
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Create user in database
	if err = db.QueryRow("INSERT INTO users (email, password, name) VALUES ($1, $2, $3) RETURNING user_id", user.Email, hashedPass, user.Name).Scan(&user.UserID); err != nil {
		if err.(*pq.Error).Code == "23505" {
			log.Printf("Register - Email already regsitered: %v\n", user.Email)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, `{"error": "Email already registered"}`, http.StatusBadRequest)
		} else {
			log.Printf("Register - Unable to insert new user into database: %v\n", err)
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}
		return
	}

	// Set user as authenticated
	session.Values["authenticated"] = true
	session.Values["name"] = user.Name
	session.Values["email"] = user.Email
	session.Values["user_id"] = user.UserID
	if err := session.Save(r, w); err != nil {
		log.Printf("Login - Unable to save session state: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}

func requestPasswordResetEmail(w http.ResponseWriter, r *http.Request) {
	// Parse and decode the request body into a new `User` instance
	user := &User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		// If there is something wrong with the request body, return a 400 status
		log.Printf("Password Reset Request - Unable to decode request body: %v\n", err)
		log.Printf("Body: %v\n", r.Body)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, `{"error": "Malformed request"}`, http.StatusBadRequest)
		return
	}

	if len(user.Email) == 0 {
		log.Println("Password Reset Request - Email not provided in reset email request")
		w.Header().Add("Content-Type", "application/json")
		SendError(w, `{"error": "Email not provided"}`, http.StatusBadRequest)
		return
	}

	// Check for the user in the database
	if err := db.QueryRow("SELECT user_id, name FROM users WHERE email = $1", user.Email).Scan(&user.UserID, &user.Name); err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Password Reset Request - Password reset requested for non-existent user %s\n", user.Email)
			w.WriteHeader(http.StatusOK)
		} else {
			log.Printf("Password Reset Request - Unable to retrieve user from database: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}
		return
	}

	// Create password reset record in database
	token := uniuri.NewLen(64)
	if _, err := db.Exec("INSERT INTO password_reset VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET user_id = $1, token = $2, expires = $3", user.UserID, token, time.Now().Add(time.Hour)); err != nil {
		log.Printf("Password Reset Request - Unable to insert password reset request into database: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Create email template
	htmlTemplate := template.Must(template.New("password_reset_email.html").ParseFiles("templates/password_reset_email.html"))
	textTemplate := template.Must(template.New("password_reset_email.txt").ParseFiles("templates/password_reset_email.txt"))

	var htmlBuffer, textBuffer bytes.Buffer
	url := "https://" + os.Getenv("HOST") + "/reset_password.html?token=" + token // + "&user_id=" + user.UserID
	data := struct {
		Href string
		Name string
	}{url, user.Name}

	if err := htmlTemplate.Execute(&htmlBuffer, data); err != nil {
		log.Printf("Password Reset Request - Unable to execute html template: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}
	if err := textTemplate.Execute(&textBuffer, data); err != nil {
		log.Printf("Password Reset Request - Unable to execute text template: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Send email
	if err := SendEmail(user.Name, user.Email, "Password Reset Email", htmlBuffer.String(), textBuffer.String()); err != nil {
		log.Printf("Password Reset Request - Failed to send password reset email: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, `{"error": "Unable to send password reset email."}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func resetPassword(w http.ResponseWriter, r *http.Request) {
	// Parse and decode the request body into a new `PasswordResetRequest` instance
	req := &PasswordResetRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		// If there is something wrong with the request body, return a 400 status
		log.Printf("Password Reset - Unable to decode request body: %v\n", err)
		log.Printf("Body: %v\n", r.Body)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, `{"error": "Malformed request"}`, http.StatusBadRequest)
		return
	}

	if len(req.Token) == 0 {
		log.Println("Password Reset - Token not provided in reset email request")
		w.Header().Add("Content-Type", "application/json")
		SendError(w, `{"error": "Token not provided"}`, http.StatusBadRequest)
		return
	}

	// Retrieve password reset request from database
	var expires time.Time
	var name, email, userID string
	if err := db.QueryRow("SELECT expires, name, email, user_id FROM password_reset JOIN users ON users.user_id = password_reset.user_id WHERE token = $1", req.Token).Scan(&expires, &name, &email); err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Password reset not found for token %s\n", req.Token)
			w.WriteHeader(http.StatusNotFound)
		} else {
			log.Printf("Password Reset - Unable to retrieve password reset request from database: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		}
		return
	}

	if expires.Before(time.Now()) {
		// Password reset request expired
		log.Printf("User %v attempt to use expired password reset, which expired on %v\n", email, expires)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, `{"error": "That password reset request has expired. Please request a new password reset email."}`, http.StatusForbidden)
		return
	}

	// User has valid password reset token. Let's reset the password!
	hashedPass, err := hashPassword(req.Password)
	if err != nil {
		log.Printf("Password Reset - Unable to hash password: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Update user in database
	if _, err = db.Exec("UPDATE users SET password = $1", hashedPass); err != nil {
		log.Printf("Reset Password - Unable to update user %v password! %v\n", email, err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Delete password request from database
	if _, err := db.Exec("DELETE FROM password_reset WHERE user_id = $1", userID); err != nil {
		log.Printf("Password Reset - Unable to clear expired credentials from database: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, DATABASE_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	// Set user as authenticated
	var session *sessions.Session
	if session, err = store.Get(r, "session"); err != nil {
		log.Printf("Password Reset - Unable to get session variables: %v\n", err)
		w.Header().Add("Content-Type", "application/json")
		SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}
	session.Values["authenticated"] = true
	session.Values["name"] = name
	session.Values["email"] = email
	session.Values["user_id"] = userID
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
			w.Header().Add("Content-Type", "application/json")
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		// Check if user is authenticated
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			log.Println("Attempt to access restricted page denied")
			SendError(w, "Forbidden", http.StatusForbidden)
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
			log.Printf("Verify Collection ID - Unable to get session: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, SERVER_ERROR_MESSAGE, http.StatusInternalServerError)
			return
		}

		acceptableIDs := session.Values["ids"].([]int64)

		// Get URL parameter
		collectionID, err := strconv.ParseInt(mux.Vars(r)["collection_id"], 10, 64)
		if err != nil {
			log.Printf("Collection ID middleware - Unable to get collection id: %v\n", err)
			w.Header().Add("Content-Type", "application/json")
			SendError(w, `{"error": "Unable to parse collection id."}`, http.StatusBadRequest)
			return
		}

		for _, id := range acceptableIDs {
			if collectionID == id {
				f(w, r)
				return
			}
		}

		log.Printf("%v | %v not found in authorized collection IDs: %v", session.Values["email"], collectionID, acceptableIDs)
		SendError(w, "Forbidden", http.StatusForbidden)
	}
}
