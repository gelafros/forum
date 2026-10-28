package handlers

import (
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"forum/internal/db"
)

type RegisterPageData struct {
	Username     string
	Email        string
	ErrorMessage string
}

// User registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check form's methods (only GET & POST allowed)
	if r.Method != http.MethodGet && r.Method != http.MethodPost {

		w.Header().Set("Allow", "GET, POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		log.Printf("RegisterHandler: method %s not allowed\n", r.Method)
		return
	}

	// Loading Registration form (method.Get)
	if r.Method == http.MethodGet {

		tmpl, err := template.ParseFiles("templates/registration_form.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("RegisterHandler template parse error:", err)
			return
		}

		err = tmpl.Execute(w, RegisterPageData{})
		if err != nil {
			log.Println("RegisterHandler template execute error:", err)
			return
		}
		return
	}

	// Reading data from registration form (method.Post)
	if r.Method == http.MethodPost {

		username := strings.TrimSpace(r.FormValue("username"))
		email := strings.TrimSpace(r.FormValue("email"))
		password := r.FormValue("password")

		// Check if ALL required fields are NOT empty
		if username == "" || email == "" || strings.TrimSpace(password) == "" {
			w.WriteHeader(http.StatusBadRequest)
			tmpl, err := template.ParseFiles("templates/registration_form.html")
			if err != nil {
				ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
				log.Println("RegisterHandler template parse error:", err)
				return
			}

			tmpl.Execute(w, RegisterPageData{
				Username:     username,
				Email:        email,
				ErrorMessage: "All fields in registration are required.",
			})
			return
		}

		// Searching DB for existing email or username
		var existingCount int

		err := db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ? OR username = ?", email, username).Scan(&existingCount)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("RegisterHandler check user error:", err)
			return
		}

		if existingCount > 0 {
			w.WriteHeader(http.StatusConflict)
			tmpl, err := template.ParseFiles("templates/registration_form.html")
			if err != nil {
				ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
				log.Println("RegisterHandler template parse error:", err)
				return
			}

			tmpl.Execute(w, RegisterPageData{
				Username:     username,
				Email:        email,
				ErrorMessage: "Username or Email is already registered.",
			})
			return
		}

		// Hash the password before storing it (never store plaintext)
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("RegisterHandler hash error:", err)
			return
		}

		// Insert New user
		_, err = db.DB.Exec("INSERT INTO users (username, email, password) VALUES (?,?,?)", username, email, string(hash))
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("RegisterHandler insert user error:", err)
			return
		}

		log.Printf("The user \"%s\" just registered to forum...", username)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
}
