package server

import (
	"log"
	"net"
	"net/http"
)

func StartServer() {
	port := ":8080"

	// Creates a file handler (any file in the folder) rooted at local folder
	http.Handle("/templates", http.StripPrefix("/templates/", http.FileServer(http.Dir("templates"))))

	// Handler for default home page "/"
	http.HandleFunc("/", HomeHandler)

	// Check the port availability for starting server in
	connection, portListerErr := net.Listen("tcp", port)
	if portListerErr != nil {
		log.Println("Error connecting in port: ", portListerErr)
		return
	} else {
		log.Println("The server is live at http://localhost:8080")
		log.Println("Press \"Ctrl + C\" in terminal for terminate the server...")
	}

	// Start the server for "connection" net.Listener
	serverErr := http.Serve(connection, nil)
	if serverErr != nil {
		log.Println("Error starting server: ", serverErr)
		return
	}
}
