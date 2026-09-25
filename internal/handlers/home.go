package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"forum/internal/db"
)

// PostSummary holds the data shown for each post on the homepage list.
type PostSummary struct {
	ID         int
	Title      string
	Author     string
	CreatedAt  string
	Categories string // comma-joined category names
}

type HomeData struct {
	Username string
	Loggedin bool
	Posts    []PostSummary
}

// loadPosts fetches all posts newest-first, joined with the author username
// and a comma-separated list of category names per post.
func loadPosts() ([]PostSummary, error) {
	rows, err := db.DB.Query(`
		SELECT
			p.id,
			p.title,
			u.username,
			p.created_at,
			COALESCE(GROUP_CONCAT(c.name, ', '), '') AS categories
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN post_categories pc ON pc.post_id = p.id
		LEFT JOIN categories c ON c.id = pc.category_id
		GROUP BY p.id
		ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []PostSummary
	for rows.Next() {
		var ps PostSummary
		if err := rows.Scan(&ps.ID, &ps.Title, &ps.Author, &ps.CreatedAt, &ps.Categories); err != nil {
			return nil, err
		}
		posts = append(posts, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

// Home Handler for starting home Forum Page
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	var (
		username string
		loggedin bool
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Method != http.MethodGet { // if method is not GET for calling home page then error
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Wrong http method detected")
		log.Println("Wrong http method detected. GET required")
		return
	}

	if r.URL.Path != "/" {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "The page you are looking for does not exist")
		return
	}

	// Check cookie to determine logged-in state
	cookie, err := r.Cookie("session_token")
	if err == nil {
		// Cookie present — validate the session against the database
		token := cookie.Value
		qErr := db.DB.QueryRow(`
			SELECT users.username
			FROM sessions
			JOIN users ON sessions.user_id = users.id
			WHERE sessions.token = ? AND sessions.expires_at > CURRENT_TIMESTAMP`, token).Scan(&username)
		if qErr == nil {
			loggedin = true
		} else if qErr != sql.ErrNoRows {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("HomeHandler session query error:", qErr)
			return
		}
	}

	// Fetch posts — visible to both guests and logged-in users
	posts, err := loadPosts()
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("HomeHandler loadPosts error:", err)
		return
	}

	data := HomeData{
		Username: username,
		Loggedin: loggedin,
		Posts:    posts,
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("HomeHandler template parse error:", err)
		return
	}
	if err = tmpl.Execute(w, data); err != nil {
		log.Println("HomeHandler template execute error:", err)
	}
}
