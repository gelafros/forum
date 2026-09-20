package server

import (
	"forum/internal/handlers"
	"log"
	"net"
	"net/http"
)

func StartServer() {
	port := ":8080"

	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)

	// Check the port availability for starting server in
	connection, err := net.Listen("tcp", port)
	if err != nil {
		log.Println("Error connecting in port: ", err)
		return
	} else {
		log.Println("The server is live at http://localhost:8080")
		log.Println("Press \"Ctrl + C\" in terminal for terminate the server...")
	}

	// Start the server for "connection" net.Listener with custom Handlefunc Router Mux
	err = http.Serve(connection, nil)
	if err != nil {
		log.Println("Error starting server: ", err)
		return
	}
}
