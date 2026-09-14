package server

import (
	"html/template"
	"net/http"
)

// Home Handler for starting home Forum Page
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; harset=utf-8")

	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		http.Error(w, "404 Not Found", http.StatusNotFound) // http Error: 404 Not Found
		return
	}

	tmpl, err := template.ParseFiles("/templates/index.html")
	if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)
}
