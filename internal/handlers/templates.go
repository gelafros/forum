package handlers

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"sync"
)

var (
	templateCache sync.Map // map[string]*template.Template
)

// getTemplate returns a cached parsed template for path, parsing it on first use.
func getTemplate(path string) (*template.Template, error) {
	if cached, ok := templateCache.Load(path); ok {
		return cached.(*template.Template), nil
	}
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		return nil, err
	}
	templateCache.Store(path, tmpl)
	return tmpl, nil
}

// renderTemplate executes a cached HTML template into an in-memory buffer before writing
// headers and status code to the http.ResponseWriter, preventing partial writes on error.
func renderTemplate(w http.ResponseWriter, r *http.Request, status int, templatePath string, data any) {
	tmpl, err := getTemplate(templatePath)
	if err != nil {
		log.Printf("renderTemplate parse error (%s): %v", templatePath, err)
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("renderTemplate execute error (%s): %v", templatePath, err)
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
