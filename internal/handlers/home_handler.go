package handlers

import (
	"html/template"
	"log"
	"net/http"
)

// Home Handler for starting home Forum Page
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Method != http.MethodGet { // if method is not GET for calling home page then error
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Wrong http method detected. GET required")
		return
	}

	if r.URL.Path != "/" {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "The page you are looking for does not exist")
		return
	}

	tmpl, err := template.ParseFiles("templates/index.html")
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
}
