package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"forum/internal/db"
)

// PostDetail holds full post data for the single-post view.
type PostDetail struct {
	ID         int
	Title      string
	Content    string
	Author     string
	CreatedAt  string
	Categories string // comma-joined category names
}

// PostDetailPageData is the data passed to post_detail.html.
type PostDetailPageData struct {
	Post     PostDetail
	Loggedin bool
	Username string
}

// PostDetailHandler serves GET /post?id=<n> and shows a single post with its details.
// Returns 404 when the post ID is missing or not found.
func PostDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		log.Printf("PostDetailHandler: method %s not allowed\n", r.Method)
		return
	}

	// Parse and validate the post ID query parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "Post ID is required.")
		return
	}
	postID, err := strconv.Atoi(idStr)
	if err != nil || postID <= 0 {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "Invalid post ID.")
		return
	}

	// Fetch the post joined with the author username and category names
	var post PostDetail
	err = db.DB.QueryRow(`
		SELECT
			p.id,
			p.title,
			p.content,
			u.username,
			p.created_at,
			COALESCE(GROUP_CONCAT(c.name, ', '), '') AS categories
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN post_categories pc ON pc.post_id = p.id
		LEFT JOIN categories c ON c.id = pc.category_id
		WHERE p.id = ?
		GROUP BY p.id`, postID).Scan(
		&post.ID,
		&post.Title,
		&post.Content,
		&post.Author,
		&post.CreatedAt,
		&post.Categories,
	)
	if err == sql.ErrNoRows {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "Post not found.")
		return
	}
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("PostDetailHandler query error:", err)
		return
	}

	// Determine logged-in state from session cookie
	var (
		username string
		loggedin bool
	)
	cookie, err := r.Cookie("session_token")
	if err == nil {
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
			log.Println("PostDetailHandler session query error:", qErr)
			return
		}
	}

	data := PostDetailPageData{
		Post:     post,
		Loggedin: loggedin,
		Username: username,
	}

	tmpl, err := template.ParseFiles("templates/post_detail.html")
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("PostDetailHandler template parse error:", err)
		return
	}
	if err = tmpl.Execute(w, data); err != nil {
		log.Println("PostDetailHandler template execute error:", err)
	}
}
