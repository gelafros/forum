package server

import (
	"log"
	"net/http"
	"time"

	"forum/internal/handlers"
)

// Helps "server_test" tester to determine the correct assciation of default mux routes (endpoints to handlers)
func registerRoutes() {
	http.HandleFunc("/templates/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/style.css")
	})

	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/auth/google/login", handlers.GoogleLoginHandler)
	http.HandleFunc("/auth/google/callback", handlers.GoogleCallbackHandler)
	http.HandleFunc("/auth/github/login", handlers.GitHubLoginHandler)
	http.HandleFunc("/auth/github/callback", handlers.GitHubCallbackHandler)
	http.HandleFunc("/create_post", handlers.CreatePostHandler)
	http.HandleFunc("/post", handlers.PostDetailHandler)
	http.HandleFunc("/post/react", handlers.PostReactHandler)
	http.HandleFunc("/comment/react", handlers.CommentReactHandler)
	http.HandleFunc("/uploads/", handlers.UploadsHandler)
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

// maxBodyBytes caps standard request bodies at 1 MiB, while /create_post allows up to 25 MiB
// so the handler can inspect and return a clear error when an image exceeds the 20 MB limit.
const (
	maxBodyBytes   = 1 << 20
	maxUploadBytes = 25 << 20
)

// limitBody rejects oversized request bodies to protect against abuse.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := int64(maxBodyBytes)
		if r.URL.Path == "/create_post" {
			limit = maxUploadBytes
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}
