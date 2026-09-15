package server

import (
	"html/template"
	"net/http"
)

type ErrorPageData struct {
	Code    int
	Title   string
	Message string
}

func ErrorPageHandler(w http.ResponseWriter, r *http.Request, status int, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("templates/errors.html")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data := ErrorPageData{
		Code:    status,
		Title:   title,
		Message: message,
	}

	w.WriteHeader(status)
	tmpl.Execute(w, data)
}
