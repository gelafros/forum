package server

import (
	"forum/internal/handlers"
	"log"
	"net"
	"net/http"
)

func StartServer() {
	port := ":8080"

	http.Handle("/templates/style.css", http.FileServer(http.Dir(".")))

	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/create_post", handlers.CreatePostHandler)

	// Check the port availability for starting server in
	connection, err := net.Listen("tcp", port)
	if err != nil {
		log.Println("Error connecting in port: ", err)
		return
	} else {
		log.Println("The server is live at http://localhost" + port)
		log.Println("Press \"Ctrl + C\" in terminal for terminate the server...")
	}

	// Start the server for "connection" net.Listener
	err = http.Serve(connection, nil)
	if err != nil {
		log.Println("Error starting server: ", err)
		return
	}
}
