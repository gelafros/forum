package server

import (
	"log"
	"net/http"

	"forum/internal/handlers"
)

// Helps "server_test" tester to determine the correct assciation of default mux routes (endpoints to handlers)
func registerRoutes() {
	http.Handle("/templates/style.css", http.FileServer(http.Dir(".")))

	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/create_post", handlers.CreatePostHandler)
	http.HandleFunc("/post", handlers.PostDetailHandler)
	http.HandleFunc("/post/react", handlers.PostReactHandler)
	http.HandleFunc("/comment/react", handlers.CommentReactHandler)
}

func StartServer() {
	registerRoutes()

	log.Println("Server is running on http://localhost:8080")
	log.Println("Press Ctrl + C to terminate the server...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
