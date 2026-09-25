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

// errInternal is the safe message shown to users for all 500 errors.
// Technical details are logged to the terminal, never sent to the browser.
const errInternal = "Something went wrong. Please try again later."

// errMethodNotAllowedTitle and errMethodNotAllowedMessage are the user-friendly
// title and explanation shown when an endpoint receives an unsupported method.
const (
	errMethodNotAllowedTitle   = "Action Not Allowed"
	errMethodNotAllowedMessage = "This action cannot be performed directly. Please use the navigation on the website."
)

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
