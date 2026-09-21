package handlers

import (
	"html/template"
	"log"
	"net/http"
)

type ErrorPageData struct {
	Code    int
	Title   string
	Message string
}

// HTML Error Handler for printing errors in browser
func ErrorPageHandler(w http.ResponseWriter, r *http.Request, status int, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("templates/errors.html")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Println("Status code: 500 - Error:", err)
		return
	}

	data := ErrorPageData{
		Code:    status,
		Title:   title,
		Message: message,
	}

	w.WriteHeader(status)
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println("Template execution failed. Error:", err)
		return
	}
}
