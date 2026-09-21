package handlers

import (
	"database/sql"
	"errors"
	"forum/internal/db"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var (
		existingUserID  int
		existingUserPsw string
	)

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Invalid http method detected")
		log.Println("Status code: 405 - Error: Method Not Allowed") // terminal print errors (technical errors report)
		return
	}

	// Load Login page
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to load login page")
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to load Login page")
			return
		}
	}

	// Proccess data from page
	if r.Method == http.MethodPost {

		username_Email := r.FormValue("identifier")
		password := r.FormValue("password")

		if username_Email == "" || password == "" {
			ErrorPageHandler(w, r, http.StatusBadRequest, "Login Failure", "Both Username/Email and Password must be completed")
			return
		}

		// Check credentials from ForumDB
		existingUserID, err := checkCredentials(w, r, username_Email, password, existingUserID, existingUserPsw)
		if err != nil {
			log.Println("Login failure error:", err)
			return
		}

		// Delete user's existing session
		_, err = db.ForumDB.Exec("DELETE FROM sessions WHERE user_id = ?", existingUserID)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to update/delete user session")
			return
		}

		// New Session token
		rawToken, err := uuid.NewRandom()
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "Error in Login procedure")
			return
		}
		token := rawToken.String()                  // Create random token
		expiresAt := time.Now().Add(24 * time.Hour) // Set session's duration for 24 Hours

		// Add token's attributes in "sessions"'s table
		createSession := "INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)"
		_, err = db.ForumDB.Exec(createSession, existingUserID, token, expiresAt)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "Session problem with existing user")
			log.Println("session failure for logged in user error:", err)
			return
		}

		// Setting coockie for browser
		cookie := &http.Cookie{
			Name:     "session_token",
			Value:    token,
			Expires:  expiresAt,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)

		// Show success message
		tmpl, err := template.ParseFiles("templates/login_success.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "An internal server error has happend")
			log.Println("Status code: 500 - Error:", err)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Println("Template execution failed. Error:", err)
			return
		}
		log.Println("A user has been Logged in to forum...")
	}
}

// Check users's credentials from ForumDB Database
func checkCredentials(w http.ResponseWriter, r *http.Request, username_Email, password string, existingUserID int, existingUserPsw string) (int, error) {
	checkDBCredentials := "SELECT id, password FROM users WHERE username = ? OR email = ?"
	err := db.ForumDB.QueryRow(checkDBCredentials, username_Email, username_Email).Scan(&existingUserID, &existingUserPsw)

	if err == sql.ErrNoRows {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Login Failure", "Invalid credentials")
		return 0, err
	}

	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "Error in Login procedure")
		return 0, err
	}

	if password != existingUserPsw { // We need this also as the above "err == sql.ErrNoRows" to be sure to check passwords after a successful .Scan
		ErrorPageHandler(w, r, http.StatusBadRequest, "Login Failure", "Invalid credentials")
		return 0, errors.New("Password don't match")
	}
	return existingUserID, nil
}
