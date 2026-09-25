package handlers

import (
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net/http"
	"time"

	"forum/internal/db"

	"github.com/google/uuid"
)

// Login user
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var (
		existingUserID  int
		existingUserPsw string
	)

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		log.Printf("LoginHandler: method %s not allowed\n", r.Method)
		return
	}

	// Load Login page
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("LoginHandler template parse error:", err)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("LoginHandler template execute error:", err)
			return
		}
	}

	// Read data from page
	if r.Method == http.MethodPost {

		username_Email := r.FormValue("identifier")
		password := r.FormValue("password")

		if username_Email == "" || password == "" {
			ErrorPageHandler(w, r, http.StatusBadRequest, "Login Failure", "Both Username/Email and Password must be completed")
			return
		}

		// Check credentials from DB
		existingUserID, err := checkCredentials(w, r, username_Email, password, existingUserID, existingUserPsw)
		if err != nil {
			log.Println("Login failure error:", err)
			return
		}

		// Delete user's existing session
		_, err = db.DB.Exec("DELETE FROM sessions WHERE user_id = ?", existingUserID)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("LoginHandler delete session error:", err)
			return
		}

		// Create new session token
		rawToken, err := uuid.NewRandom() // Create random token
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("LoginHandler uuid error:", err)
			return
		}
		token := rawToken.String()                  // Transpose uuid token to string
		expiresAt := time.Now().Add(24 * time.Hour) // Set session's duration for 24 Hours

		// Add token's attributes in "sessions"'s table
		createSession := "INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)"
		_, err = db.DB.Exec(createSession, existingUserID, token, expiresAt)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("LoginHandler insert session error:", err)
			return
		}

		// Create cookie for browser
		cookie := &http.Cookie{
			Name:     "session_token",
			Value:    token,
			Expires:  expiresAt,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)

		// Show success Login message
		tmpl, err := template.ParseFiles("templates/login_success.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("LoginHandler success template parse error:", err)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Println("LoginHandler success template execute error:", err)
			return
		}
		log.Println("A user has been Logged in to the forum...")
	}
}

// Check users's credentials from DB Database
func checkCredentials(w http.ResponseWriter, r *http.Request, username_Email, password string, existingUserID int, existingUserPsw string) (int, error) {
	checkDBCredentials := "SELECT id, password FROM users WHERE username = ? OR email = ?"
	err := db.DB.QueryRow(checkDBCredentials, username_Email, username_Email).Scan(&existingUserID, &existingUserPsw)

	if err == sql.ErrNoRows {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Login Failure", "Invalid credentials")
		return 0, err
	}

	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("checkCredentials query error:", err)
		return 0, err
	}

	if password != existingUserPsw { // We need this also as the above "err == sql.ErrNoRows" to be sure to check passwords after a successful .Scan
		ErrorPageHandler(w, r, http.StatusBadRequest, "Login Failure", "Invalid credentials")
		return 0, errors.New("Password don't match")
	}
	return existingUserID, nil
}
