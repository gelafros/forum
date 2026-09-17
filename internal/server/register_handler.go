package server

import (
	"database/sql"
	"html/template"
	"net/http"
)

type Data struct {
	username string
	email    string
}

func RegisterHandler(w http.ResponseWriter, r *http.Request, forumDB *sql.DB) {
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
			ErrorPageHandler(w, r, http.StatusInternalServerError, "500 Internal Server Error", "An internal server error has happend")
			return
		}

		tmpl.Execute(w, nil)
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

		tmpl, err := template.ParseFiles("templates/registration_form.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "500 Internal Server Error", "An internal server error has happend")
			return
		}

		tmpl.Execute(w, nil)
	}
}
