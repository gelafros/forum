package server

import (
	"log"
	"net/http"
	"time"

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

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           limitBody(http.DefaultServeMux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

// maxBodyBytes caps request bodies (posts, comments, forms) at 1 MiB.
const maxBodyBytes = 1 << 20

// limitBody rejects oversized request bodies to protect against abuse.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}
