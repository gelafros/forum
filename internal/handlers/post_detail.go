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
	ID             int
	Title          string
	Content        string
	Author         string
	AuthorInitial  string
	CreatedAt      string
	Categories     string // comma-joined category names
	CategoriesList []string
	Likes          int
	Dislikes       int
	UserReaction   string // "like", "dislike", or ""
	ImageURL       string
}

// Comment holds comment data displayed under a post.
type Comment struct {
	ID            int
	Content       string
	Author        string
	AuthorInitial string
	CreatedAt     string
	Likes         int
	Dislikes      int
	UserReaction  string
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
			datetime(p.created_at, 'localtime') AS created_at,
			COALESCE(GROUP_CONCAT(c.name, ', '), '') AS categories,
			COALESCE(pi.image_path, '') AS image_path
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN post_categories pc ON pc.post_id = p.id
		LEFT JOIN categories c ON c.id = pc.category_id
		LEFT JOIN post_images pi ON pi.post_id = p.id
		WHERE p.id = ?
		GROUP BY p.id`, postID).Scan(
		&post.ID,
		&post.Title,
		&post.Content,
		&post.Author,
		&post.CreatedAt,
		&post.Categories,
		&post.ImageURL,
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

	// Count Likes/Dislikes
	// Hold in "row" the SUM of "like" and "dislike" for a specific post (postID)
	row := db.DB.QueryRow(`SELECT
							COUNT(CASE WHEN reaction_type = 'like' THEN 1 END),
							COUNT(CASE WHEN reaction_type = 'dislike' THEN 1 END)
						FROM post_reactions
						WHERE post_id = ?`, postID)

	err = row.Scan(&post.Likes, &post.Dislikes) // add SUM's values in the specific fields
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("PostDetailHandler likes/dislikes counter error:", err)
		return
	}

	// Determine logged-in state from session cookie
	session, err := CheckSession(r)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	// Query user reaction if logged in
	if session.Loggedin {
		_ = db.DB.QueryRow("SELECT reaction_type FROM post_reactions WHERE user_id = ? AND post_id = ?", session.UserID, postID).Scan(&post.UserReaction)
	}

	if len(post.Author) > 0 {
		runes := []rune(post.Author)
		post.AuthorInitial = strings.ToUpper(string(runes[0]))
	}

	if post.Categories != "" {
		for _, cat := range strings.Split(post.Categories, ",") {
			cat = strings.TrimSpace(cat)
			if cat != "" {
				post.CategoriesList = append(post.CategoriesList, cat)
			}
		}
	}

	// Load comments for this post (SELECT COUNT(*) is an extra insider selection and * = for all rows)
	commentRows, err := db.DB.Query(`
		SELECT
			c.id,
			c.content,
			u.username,
			datetime(c.created_at, 'localtime') AS created_at,
			(SELECT COUNT(*) FROM comment_reactions cr
			 WHERE cr.comment_id = c.id AND cr.reaction_type = 'like') AS likes,
			(SELECT COUNT(*) FROM comment_reactions cr
			 WHERE cr.comment_id = c.id AND cr.reaction_type = 'dislike') AS dislikes,
			COALESCE((SELECT reaction_type FROM comment_reactions cr
			 WHERE cr.comment_id = c.id AND cr.user_id = ?), '') AS user_reaction
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC, c.id ASC`, session.UserID, postID)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("PostDetailHandler comments query error:", err)
		return
	}
	defer commentRows.Close()

	// List of comments
	var comments []Comment
	for commentRows.Next() {
		var cm Comment
		if err := commentRows.Scan(&cm.ID, &cm.Content, &cm.Author, &cm.CreatedAt, &cm.Likes, &cm.Dislikes, &cm.UserReaction); err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("PostDetailHandler comments scan error:", err)
			return
		}
		if len(cm.Author) > 0 {
			runes := []rune(cm.Author)
			cm.AuthorInitial = strings.ToUpper(string(runes[0]))
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
			Post:         post,
			Comments:     comments,
			Loggedin:     session.Loggedin,
			Username:     session.Username,
			ErrorMessage: "",
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
	if !session.Loggedin {
		ErrorPageHandler(w, r, http.StatusUnauthorized, "401 Unauthorized", "You must be logged in to create a comment.")
		return
	}

	commentContent := strings.TrimSpace(r.FormValue("content"))

	// Empty comment case
	if commentContent == "" {

		data := PostDetailPageData{
			Post:         post,
			Comments:     comments,
			Loggedin:     session.Loggedin,
			Username:     session.Username,
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

	_, err = db.DB.Exec("INSERT INTO comments (user_id, post_id, content) VALUES (?, ?, ?)", session.UserID, postID, commentContent)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("PostDetailHandler insert comment error:", err)
		return
	}

	http.Redirect(w, r, "/post?id="+strconv.Itoa(postID), http.StatusSeeOther)
}
