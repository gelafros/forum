package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"forum/internal/db"
)

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	var existingUserID int

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		log.Printf("LogoutHandler: method %s not allowed\n", r.Method)
		return
	}

	// Read from page
	if r.Method == http.MethodPost {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			log.Println("Cookie problem in logging out user")
			return
		}
		token := cookie.Value
		err = db.DB.QueryRow("SELECT user_id FROM sessions WHERE sessions.token = ?", token).Scan(&existingUserID)

		if err == sql.ErrNoRows {
			// Session token not found — treat as already logged out, redirect home
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("LogoutHandler session query error:", err)
			return
		}

		// Delete user's existing session
		_, err = db.DB.Exec("DELETE FROM sessions WHERE user_id = ?", existingUserID)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("LogoutHandler delete session error:", err)
			return
		}

		// Delete cookie from browser
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		// Redirect to HomePage ("/")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
}
