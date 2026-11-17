package handlers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"forum/internal/db"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginPageData struct {
	Identifier   string
	ErrorMessage string
}

// Login user
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		log.Printf("LoginHandler: method %s not allowed\n", r.Method)
		return
	}

	if r.Method == http.MethodGet {
		renderTemplate(w, r, http.StatusOK, "templates/login.html", LoginPageData{})
		return
	}

	usernameEmail := r.FormValue("identifier")
	password := r.FormValue("password")

	if usernameEmail == "" || password == "" {
		renderTemplate(w, r, http.StatusBadRequest, "templates/login.html", LoginPageData{
			Identifier:   usernameEmail,
			ErrorMessage: "Both Username/Email and Password must be completed.",
		})
		return
	}

	existingUserID, err := checkCredentials(usernameEmail, password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "invalid credentials" {
			renderTemplate(w, r, http.StatusUnauthorized, "templates/login.html", LoginPageData{
				Identifier:   usernameEmail,
				ErrorMessage: "Invalid username/email or password.",
			})
			return
		}
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("Login failure error:", err)
		return
	}

	if err := issueUserSession(w, existingUserID); err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("LoginHandler session error:", err)
		return
	}

	log.Printf("The user %q has just logged in...", usernameEmail)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// issueUserSession replaces any existing session for userID with a new 24h UUID session token and sets the HttpOnly cookie.
func issueUserSession(w http.ResponseWriter, userID int) error {
	rawToken, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	token := rawToken.String()
	expiresAt := time.Now().Add(24 * time.Hour)

	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM sessions WHERE user_id = ?", userID); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)", userID, token, expiresAt); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("COOKIE_SECURE") == "true",
	})
	return nil
}

// checkCredentials validates user's credentials against SQLite and returns user's ID.
func checkCredentials(usernameEmail, password string) (int, error) {
	var (
		existingUserID  int
		existingUserPsw string
	)

	err := db.DB.QueryRow("SELECT id, password FROM users WHERE username = ? OR email = ?", usernameEmail, usernameEmail).Scan(&existingUserID, &existingUserPsw)
	if err != nil {
		return 0, err
	}

	if bcrypt.CompareHashAndPassword([]byte(existingUserPsw), []byte(password)) != nil {
		return 0, errors.New("invalid credentials")
	}
	return existingUserID, nil
}
