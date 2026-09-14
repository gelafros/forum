package server

import (
	"log"
	"net/http"
)

func StartServer() error {
	// Creates a file handler (any file in the folder) rooted at local folder
	http.Handle("/templates", http.StripPrefix("/templates/", http.FileServer(http.Dir("templates"))))

	// Handler for default home page :8080/
	http.HandleFunc("/", HomeHandler)

	// Start the server @8080
	log.Println("The server is live at http://localhost:8080")
	log.Println("Press \"Ctrl + C\" in terminal for terminate the server...")
	http.ListenAndServe(":8080", nil)
}
