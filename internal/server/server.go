package server

import (
	"database/sql"
	"log"
	"net"
	"net/http"
)

func StartServer(forumDB *sql.DB) {
	port := ":8080"

	// Custom router handler (for multiple routes)
	mux := http.NewServeMux()
	RegisterRoutes(forumDB, mux)

	// Check the port availability for starting server in
	connection, portListenErr := net.Listen("tcp", port)
	if portListenErr != nil {
		log.Println("Error connecting in port: ", portListenErr)
		return
	} else {
		log.Println("The server is live at http://localhost:8080")
		log.Println("Press \"Ctrl + C\" in terminal for terminate the server...")
	}

	// Start the server for "connection" net.Listener with custom Handlefunc Router Mux
	serverErr := http.Serve(connection, mux)
	if serverErr != nil {
		log.Println("Error starting server: ", serverErr)
		return
	}
}
