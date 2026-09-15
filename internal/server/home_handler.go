package server

import (
	"html/template"
	"net/http"
)

// Home Handler for starting home Forum Page
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Method != http.MethodGet { // if method is not GET for calling home page then error
		// http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Wrong http method detected. GET required")
		return
	}

	if r.URL.Path != "/" {
		// http.Error(w, "404 Not Found", http.StatusNotFound) // http Error: 404 Not Found
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "The page you are looking for does not exist")
		return
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		// http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		ErrorPageHandler(w, r, http.StatusInternalServerError, "500 Internal Server Error", "An internal server error has happend")
		return
	}

	tmpl.Execute(w, nil)
}
