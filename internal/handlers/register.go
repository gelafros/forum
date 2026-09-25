package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"forum/internal/db"
)

type Data struct {
	username string
	email    string
}

// User registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var existingUserID int

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check form's methods (only GET & POST allowed)
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		log.Printf("RegisterHandler: method %s not allowed\n", r.Method)
		return
	}

	// Loading Rgistration from (method.Get)
	if r.Method == http.MethodGet {

		tmpl, err := template.ParseFiles("templates/registration_form.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("RegisterHandler template parse error:", err)
			return
		}

		err = tmpl.Execute(w, nil)
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
			ErrorPageHandler(w, r, http.StatusBadRequest, "Registration Failure", "All fields in registration are required")
			return
		}

		// Searching DB for existing email
		searchForumByEmail := "SELECT id FROM users WHERE email = ?"
		err := db.DB.QueryRow(searchForumByEmail, email).Scan(&existingUserID)

		// 1. New Email Registration
		if err == sql.ErrNoRows {

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
			return // End when succesfully register a New user
		}

		// 2. Email duplicate or Error
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("RegisterHandler email query error:", err)
			return
		} else {
			ErrorPageHandler(w, r, http.StatusConflict, "Conflict Error", "This email already belongs to another user")
			return
		}
	}
}
