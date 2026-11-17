package handlers

import (
	"log"
	"net/http"
	"strings"

	"forum/internal/db"

	"golang.org/x/crypto/bcrypt"
)

type RegisterPageData struct {
	Username     string
	Email        string
	ErrorMessage string
}

// User registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		log.Printf("RegisterHandler: method %s not allowed\n", r.Method)
		return
	}

	if r.Method == http.MethodGet {
		renderTemplate(w, r, http.StatusOK, "templates/registration_form.html", RegisterPageData{})
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if username == "" || email == "" || strings.TrimSpace(password) == "" {
		renderTemplate(w, r, http.StatusBadRequest, "templates/registration_form.html", RegisterPageData{
			Username:     username,
			Email:        email,
			ErrorMessage: "All fields in registration are required.",
		})
		return
	}

	var existingCount int
	if err := db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ? OR username = ?", email, username).Scan(&existingCount); err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("RegisterHandler check user error:", err)
		return
	}

	if existingCount > 0 {
		renderTemplate(w, r, http.StatusConflict, "templates/registration_form.html", RegisterPageData{
			Username:     username,
			Email:        email,
			ErrorMessage: "Username or Email is already registered.",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("RegisterHandler hash error:", err)
		return
	}

	if _, err := db.DB.Exec("INSERT INTO users (username, email, password) VALUES (?,?,?)", username, email, string(hash)); err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("RegisterHandler insert user error:", err)
		return
	}

	log.Printf("The user %q just registered to forum...", username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
