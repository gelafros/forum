package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"forum/internal/db"
)

type HomeData struct {
	Username string
	Loggedin bool
}

// Home Handler for starting home Forum Page
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	var (
		username string
		loggedin bool
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Method != http.MethodGet { // if method is not GET for calling home page then error
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Wrong http method detected")
		log.Println("Wrong http method detected. GET required")
		return
	}

	if r.URL.Path != "/" {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "The page you are looking for does not exist")
		return
	}

	// Check cookie for show login Username
	cookie, err := r.Cookie("session_token")
	// Render the Home page as Guest
	if err != nil {
		loggedin = false
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "An internal server error has happend")
			log.Println("Status code: 500 - Error:", err) // terminal print errors (technical errors report)
			return
		}
		data := HomeData{
			Username: username,
			Loggedin: false,
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Println("Template execution failed. Error:", err)
			return
		}
		return
	}

	token := cookie.Value
	err = db.DB.QueryRow(`
		SELECT users.username
		FROM sessions
		JOIN users ON sessions.user_id = users.id
		WHERE sessions.token = ? AND sessions.expires_at > CURRENT_TIMESTAMP`, token).Scan(&username)

	// Render the Home page as Guest
	if err == sql.ErrNoRows {
		loggedin = false
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "An internal server error has happend")
			log.Println("Status code: 500 - Error:", err) // terminal print errors (technical errors report)
			return
		}
		data := HomeData{
			Username: username,
			Loggedin: false,
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Println("Template execution failed. Error:", err)
			return
		}
		return
	}
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "An internal server error has happend in checking cookies")
		log.Println("Status code: 500 - Error:", err) // terminal print errors (technical errors report)
		return
	}

	// Render the Home page as loggedin user
	loggedin = true
	if loggedin == true {
		data := HomeData{
			Username: username,
			Loggedin: true,
		}
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", "An internal server error has happend")
			log.Println("Status code: 500 - Error:", err) // terminal print errors (technical errors report)
			return
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Println("Template execution failed. Error:", err)
			return
		}
	}
}
