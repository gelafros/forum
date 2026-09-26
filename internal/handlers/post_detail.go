package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

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

// Comment holds comment data displayed under a post.
type Comment struct {
	ID        int
	Content   string
	Author    string
	CreatedAt string
}

// PostDetailPageData is the data passed to post_detail.html.
type PostDetailPageData struct {
	Post         PostDetail
	Comments     []Comment
	Loggedin     bool
	Username     string
	ErrorMessage string
}

// PostDetailHandler serves GET /post?id=<n> and POST /post?id=<n>.
// GET shows a single post with its comments.
// POST creates a new comment under the post (authenticated users only).
func PostDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
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
		userID   int
		username string
		loggedin bool
	)
	cookie, err := r.Cookie("session_token")
	if err == nil {
		token := cookie.Value
		qErr := db.DB.QueryRow(`
			SELECT users.id, users.username
			FROM sessions
			JOIN users ON sessions.user_id = users.id
			WHERE sessions.token = ? AND sessions.expires_at > CURRENT_TIMESTAMP`, token).Scan(&userID, &username)
		if qErr == nil {
			loggedin = true
		} else if qErr != sql.ErrNoRows {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("PostDetailHandler session query error:", qErr)
			return
		}
	}

	// Load comments for this post
	commentRows, err := db.DB.Query(`
		SELECT
			c.id,
			c.content,
			u.username,
			c.created_at
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC, c.id ASC`, postID)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("PostDetailHandler comments query error:", err)
		return
	}
	defer commentRows.Close()

	var comments []Comment
	for commentRows.Next() {
		var cm Comment
		if err := commentRows.Scan(&cm.ID, &cm.Content, &cm.Author, &cm.CreatedAt); err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("PostDetailHandler comments scan error:", err)
			return
		}
		comments = append(comments, cm)
	}
	if err := commentRows.Err(); err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("PostDetailHandler comments rows error:", err)
		return
	}

	if r.Method == http.MethodGet {

		data := PostDetailPageData{
			Post:     post,
			Comments: comments,
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
		return
	}

	// POST — create comment
	if !loggedin {
		ErrorPageHandler(w, r, http.StatusUnauthorized, "401 Unauthorized", "You must be logged in to create a comment.")
		return
	}

	commentContent := strings.TrimSpace(r.FormValue("content"))
	if commentContent == "" {

		data := PostDetailPageData{
			Post:         post,
			Comments:     comments,
			Loggedin:     loggedin,
			Username:     username,
			ErrorMessage: "Comment content cannot be empty.",
		}

		w.WriteHeader(http.StatusBadRequest)
		tmpl, err := template.ParseFiles("templates/post_detail.html")
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("PostDetailHandler template parse error:", err)
			return
		}

		if err = tmpl.Execute(w, data); err != nil {
			log.Println("PostDetailHandler template execute error:", err)
		}
		return
	}

	_, err = db.DB.Exec("INSERT INTO comments (user_id, post_id, content) VALUES (?, ?, ?)", userID, postID, commentContent)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("PostDetailHandler insert comment error:", err)
		return
	}

	http.Redirect(w, r, "/post?id="+strconv.Itoa(postID), http.StatusSeeOther)
}
