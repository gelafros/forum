package handlers

import (
	"database/sql"
	"forum/internal/db"
	"html/template"
	"log"
	"net/http"
)

type Data struct {
	username string
	email    string
}

// User registration
// func RegisterHandler(w http.ResponseWriter, r *http.Request, forumDB *sql.DB) {
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var existingUserID int

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check form's methods (only GET & POST allowed)
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Wrong http method detected")
		return
	}

	// Loading Rgistration from (method.Get)
	if r.Method == http.MethodGet {

		if r.URL.Path != "/register" {
			ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "The page you are looking for does not exist")
			return
		}

		tmpl, err := template.ParseFiles("templates/registration_form.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "An internal server error has happend")
			log.Println("Status code: 500 - Error:", err) // terminal print errors (technical errors report)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Println("Template execution failed. Error:", err)
			return
		}
		return
	}

	// Reading data from registration form (method.Post)
	if r.Method == http.MethodPost {
		formUsername := r.FormValue("username")
		formEmail := r.FormValue("email")
		formPassword := r.FormValue("password")

		// Check if ALL required fields are NOT empty
		if formUsername == "" || formEmail == "" || formPassword == "" {
			ErrorPageHandler(w, r, http.StatusBadRequest, "Registration Failure", "All fields in registration are required")
			return
		}

		// Searching forumDB for existing email
		searchForumByEmail := "SELECT id FROM users WHERE email = ?"
		err := db.ForumDB.QueryRow(searchForumByEmail, formEmail).Scan(&existingUserID)

		// 1. New Email Registration
		if err == sql.ErrNoRows {

			insertNewUser := `INSERT INTO users (username, email, password) VALUES (?,?,?)`
			_, err = db.ForumDB.Exec(insertNewUser, formUsername, formEmail, formPassword)
			if err != nil {
				ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "New user registration failed")
				log.Println("New user registration failed err:", err)
				return
			}

			// Show Success Message
			tmpl, err := template.ParseFiles("templates/registration_success.html")
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
			log.Println("New User has been registered to forum...")
			return // End when succesfully register a New user
		}

		// 2. Email duplicate or Error
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "Email query from ForumDB has Failed")
			log.Println("Error in Email query from ForumDB:", err)
			return
		} else {
			ErrorPageHandler(w, r, http.StatusConflict, "Conflict Error", "This Email is already belongs to another user")
			return
		}
	}
}
