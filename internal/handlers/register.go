package handlers

import (
	"html/template"
	"log"
	"net/http"

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
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Check if ALL required fields are NOT empty
		if username == "" || email == "" || password == "" {
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

		// Insert new user
		insertNewUser := `INSERT INTO users (username, email, password) VALUES (?,?,?)`
		_, err = db.DB.Exec(insertNewUser, username, email, password)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("RegisterHandler insert user error:", err)
			return
		}

		// Show Success Message
		tmpl, err := template.ParseFiles("templates/registration_success.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("RegisterHandler success template parse error:", err)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Println("RegisterHandler success template execute error:", err)
			return
		}
		log.Println("New User has been registered to forum...")
		return
	}
}
