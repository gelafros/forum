package server

import (
	"log"
	"net/http"
	"os"
	"time"

	"forum/internal/handlers"
)

// Helps "server_test" tester to determine the correct association of default mux routes (endpoints to handlers)
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
	http.HandleFunc("/moderation", handlers.ModerationDashboardHandler)
	http.HandleFunc("/moderation/request-role", handlers.RequestModeratorRoleHandler)
	http.HandleFunc("/moderation/role", handlers.ManageRoleHandler)
	http.HandleFunc("/moderation/post/approve", handlers.ApprovePostHandler)
	http.HandleFunc("/moderation/post/delete", handlers.DeletePostModerationHandler)
	http.HandleFunc("/moderation/comment/delete", handlers.DeleteCommentModerationHandler)
	http.HandleFunc("/moderation/report", handlers.ReportPostHandler)
	http.HandleFunc("/moderation/report/respond", handlers.RespondReportHandler)
	http.HandleFunc("/moderation/categories", handlers.ManageCategoriesHandler)
}

func StartServer() {
	registerRoutes()

	limiter := NewRateLimiter(20, 40)
	defer limiter.Close()

	handler := SecurityHeadersMiddleware(limiter.Middleware(limitBody(http.DefaultServeMux)))

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		TLSConfig:         NewSecureTLSConfig(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if os.Getenv("ENABLE_HTTPS") == "true" {
		certFile := os.Getenv("TLS_CERT_FILE")
		if certFile == "" {
			certFile = "certs/cert.pem"
		}
		keyFile := os.Getenv("TLS_KEY_FILE")
		if keyFile == "" {
			keyFile = "certs/key.pem"
		}
		if err := EnsureTLSCertificate(certFile, keyFile); err != nil {
			log.Fatalf("Failed to initialize TLS certificate: %v", err)
		}
		_ = os.Setenv("COOKIE_SECURE", "true")
		log.Printf("HTTPS server is running on https://localhost%s\n", addr)
		log.Println("Press Ctrl + C to terminate the server...")
		log.Fatal(srv.ListenAndServeTLS(certFile, keyFile))
		return
	}

	log.Printf("Server is running on http://localhost%s\n", addr)
	log.Println("Press Ctrl + C to terminate the server...")
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
