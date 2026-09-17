package server

import (
	"database/sql"
	"net/http"
)

func RegisterRoutes(forumDB *sql.DB, mux *http.ServeMux) {

	// Creates a file handler (any file in the folder) rooted at local folder
	mux.Handle("/templates", http.StripPrefix("/templates/", http.FileServer(http.Dir("templates"))))

	// Route to endpoints
	mux.HandleFunc("/", HomeHandler)
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) { RegisterHandler(w, r, forumDB) }) // wrapper (anonymous func) passes the Database's connection object (forumDB) data to RegisterHandler
}
