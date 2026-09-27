package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/db"
)

// PostSummary holds the data shown for each post on the homepage list.
type PostSummary struct {
	ID            int
	Title         string
	Content       string
	Author        string
	AuthorInitial string
	CreatedAt     string
	Categories    []string
	Likes         int
	Dislikes      int
	CommentCount  int
}

type HomeData struct {
	Username       string
	Loggedin       bool
	Posts          []PostSummary
	Categories     []Category
	ActiveCategory int
	ActiveFilter   string
}

// loadPosts fetches posts newest-first with snippet, counts, and optional category/user filters.
func loadPosts(categoryID int, filter string, userID int) ([]PostSummary, error) {
	baseQuery := `
		SELECT
			p.id,
			p.title,
			p.content,
			u.username,
			datetime(p.created_at, 'localtime') AS created_at,
			COALESCE(GROUP_CONCAT(DISTINCT c.name), '') AS categories,
			(SELECT COUNT(*) FROM post_reactions WHERE post_id = p.id AND reaction_type = 'like') AS likes,
			(SELECT COUNT(*) FROM post_reactions WHERE post_id = p.id AND reaction_type = 'dislike') AS dislikes,
			(SELECT COUNT(*) FROM comments WHERE post_id = p.id) AS comment_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN post_categories pc ON pc.post_id = p.id
		LEFT JOIN categories c ON c.id = pc.category_id`

	var joins []string
	var wheres []string
	var args []any

	if categoryID > 0 {
		joins = append(joins, "JOIN post_categories fpc ON fpc.post_id = p.id AND fpc.category_id = ?")
		args = append(args, categoryID)
	}

	if filter == "created" && userID > 0 {
		wheres = append(wheres, "p.user_id = ?")
		args = append(args, userID)
	} else if filter == "liked" && userID > 0 {
		joins = append(joins, "JOIN post_reactions fpr ON fpr.post_id = p.id AND fpr.user_id = ? AND fpr.reaction_type = 'like'")
		args = append(args, userID)
	}

	query := baseQuery
	for _, j := range joins {
		query += " " + j
	}
	if len(wheres) > 0 {
		query += " WHERE " + strings.Join(wheres, " AND ")
	}
	query += " GROUP BY p.id ORDER BY p.created_at DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []PostSummary
	for rows.Next() {
		var ps PostSummary
		var catStr string
		if err := rows.Scan(&ps.ID, &ps.Title, &ps.Content, &ps.Author, &ps.CreatedAt, &catStr, &ps.Likes, &ps.Dislikes, &ps.CommentCount); err != nil {
			return nil, err
		}
		if catStr != "" {
			for _, c := range strings.Split(catStr, ",") {
				c = strings.TrimSpace(c)
				if c != "" {
					ps.Categories = append(ps.Categories, c)
				}
			}
		}
		if len(ps.Author) > 0 {
			runes := []rune(ps.Author)
			ps.AuthorInitial = strings.ToUpper(string(runes[0]))
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		log.Printf("HomeHandler: method %s not allowed\n", r.Method)
		return
	}

	if r.URL.Path != "/" {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "The page you are looking for does not exist")
		return
	}

	// Check cookie to determine logged-in state
	session, err := CheckSession(r)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	// Read category or user filter from query parameters
	categoryStr := r.URL.Query().Get("category")
	filterStr := r.URL.Query().Get("filter")

	categoryID, _ := strconv.Atoi(categoryStr)

	// Guard: guests cannot filter by created or liked
	if !session.Loggedin && (filterStr == "created" || filterStr == "liked") {
		filterStr = ""
	}

	// Fetch posts with filter applied
	posts, err := loadPosts(categoryID, filterStr, session.UserID)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("HomeHandler loadPosts error:", err)
		return
	}

	categories := LoadCategories()

	data := HomeData{
		Username:       session.Username,
		Loggedin:       session.Loggedin,
		Posts:          posts,
		Categories:     categories,
		ActiveCategory: categoryID,
		ActiveFilter:   filterStr,
	}

	// Render page (index.html)
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("HomeHandler template parse error:", err)
		return
	}

	// Seed the page with data
	if err = tmpl.Execute(w, data); err != nil {
		log.Println("HomeHandler template execute error:", err)
	}
}
