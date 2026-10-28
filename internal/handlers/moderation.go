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

var validReportReasons = map[string]bool{
	"irrelevant": true,
	"obscene":    true,
	"illegal":    true,
	"insulting":  true,
}

type RoleRequestItem struct {
	ID        int
	UserID    int
	Username  string
	Status    string
	CreatedAt string
}

type ModerationReportItem struct {
	ID            int
	PostID        int
	PostTitle     string
	Reporter      string
	Reason        string
	Details       string
	Status        string
	AdminResponse string
	CreatedAt     string
}

type PendingPostItem struct {
	ID        int
	Title     string
	Content   string
	Author    string
	CreatedAt string
}

type UserRoleItem struct {
	ID       int
	Username string
	Email    string
	Role     string
}

type ModerationDashboardData struct {
	Session       Session
	Categories    []Category
	PendingPosts  []PendingPostItem
	Reports       []ModerationReportItem
	RoleRequests  []RoleRequestItem
	Users         []UserRoleItem
	MyRoleRequest string
	Message       string
}

// DeletePostByID removes a post and all dependent rows inside a transaction.
func DeletePostByID(postID int) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queries := []string{
		"DELETE FROM comment_reactions WHERE comment_id IN (SELECT id FROM comments WHERE post_id = ?)",
		"DELETE FROM comments WHERE post_id = ?",
		"DELETE FROM post_reactions WHERE post_id = ?",
		"DELETE FROM post_categories WHERE post_id = ?",
		"DELETE FROM post_images WHERE post_id = ?",
		"DELETE FROM post_moderation WHERE post_id = ?",
		"DELETE FROM moderation_reports WHERE post_id = ?",
		"DELETE FROM posts WHERE id = ?",
	}
	for _, q := range queries {
		if _, err := tx.Exec(q, postID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ModerationDashboardHandler renders GET /moderation for users, moderators, and administrators.
func ModerationDashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}

	session, err := CheckSession(r)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}
	if !session.Loggedin {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := ModerationDashboardData{
		Session:    session,
		Categories: LoadCategories(),
		Message:    strings.TrimSpace(r.URL.Query().Get("msg")),
	}

	_ = db.DB.QueryRow("SELECT status FROM role_requests WHERE user_id = ?", session.UserID).Scan(&data.MyRoleRequest)

	if session.Role == "moderator" || session.Role == "admin" {
		rows, err := db.DB.Query(`
			SELECT p.id, p.title, p.content, u.username, datetime(p.created_at, 'localtime')
			FROM posts p
			JOIN users u ON u.id = p.user_id
			JOIN post_moderation pm ON pm.post_id = p.id
			WHERE pm.status = 'pending'
			ORDER BY p.created_at ASC`)
		if err == nil {
			for rows.Next() {
				var item PendingPostItem
				if err := rows.Scan(&item.ID, &item.Title, &item.Content, &item.Author, &item.CreatedAt); err == nil {
					data.PendingPosts = append(data.PendingPosts, item)
				}
			}
			rows.Close()
		}

		repRows, err := db.DB.Query(`
			SELECT mr.id, mr.post_id, COALESCE(p.title, '[Deleted Post]'), u.username, mr.reason, mr.details, mr.status, mr.admin_response, datetime(mr.created_at, 'localtime')
			FROM moderation_reports mr
			JOIN users u ON u.id = mr.reporter_id
			LEFT JOIN posts p ON p.id = mr.post_id
			ORDER BY mr.created_at DESC`)
		if err == nil {
			for repRows.Next() {
				var rep ModerationReportItem
				if err := repRows.Scan(&rep.ID, &rep.PostID, &rep.PostTitle, &rep.Reporter, &rep.Reason, &rep.Details, &rep.Status, &rep.AdminResponse, &rep.CreatedAt); err == nil {
					data.Reports = append(data.Reports, rep)
				}
			}
			repRows.Close()
		}
	}

	if session.Role == "admin" {
		reqRows, err := db.DB.Query(`
			SELECT rr.id, rr.user_id, u.username, rr.status, datetime(rr.created_at, 'localtime')
			FROM role_requests rr
			JOIN users u ON u.id = rr.user_id
			WHERE rr.status = 'pending'
			ORDER BY rr.created_at ASC`)
		if err == nil {
			for reqRows.Next() {
				var rr RoleRequestItem
				if err := reqRows.Scan(&rr.ID, &rr.UserID, &rr.Username, &rr.Status, &rr.CreatedAt); err == nil {
					data.RoleRequests = append(data.RoleRequests, rr)
				}
			}
			reqRows.Close()
		}

		userRows, err := db.DB.Query(`
			SELECT u.id, u.username, u.email, COALESCE(ur.role, 'user')
			FROM users u
			LEFT JOIN user_roles ur ON ur.user_id = u.id
			ORDER BY u.id ASC`)
		if err == nil {
			for userRows.Next() {
				var u UserRoleItem
				if err := userRows.Scan(&u.ID, &u.Username, &u.Email, &u.Role); err == nil {
					data.Users = append(data.Users, u)
				}
			}
			userRows.Close()
		}
	}

	tmpl, err := template.ParseFiles("templates/moderation.html")
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("ModerationDashboardHandler template parse error:", err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

// RequestModeratorRoleHandler allows a registered user to request promotion to moderator from an admin.
func RequestModeratorRoleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}
	session, err := CheckSession(r)
	if err != nil || !session.Loggedin {
		ErrorPageHandler(w, r, http.StatusUnauthorized, "401 Unauthorized", "You must be logged in to request a moderator role.")
		return
	}
	if session.Role != "user" {
		ErrorPageHandler(w, r, http.StatusBadRequest, "400 Bad Request", "Only standard users can request promotion to moderator.")
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO role_requests (user_id, status) VALUES (?, 'pending')
		ON CONFLICT(user_id) DO UPDATE SET status = 'pending', created_at = CURRENT_TIMESTAMP`,
		session.UserID,
	)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}
	http.Redirect(w, r, "/moderation?msg=Role+request+submitted", http.StatusSeeOther)
}

// ManageRoleHandler allows an administrator to promote or demote users (user <-> moderator).
func ManageRoleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}
	session, err := CheckSession(r)
	if err != nil || !session.Loggedin || session.Role != "admin" {
		ErrorPageHandler(w, r, http.StatusForbidden, "403 Forbidden", "Administrator access is required to manage user roles.")
		return
	}

	targetUserID, err := strconv.Atoi(r.FormValue("user_id"))
	newRole := strings.TrimSpace(r.FormValue("role"))
	if err != nil || targetUserID <= 0 || (newRole != "user" && newRole != "moderator" && newRole != "admin") {
		ErrorPageHandler(w, r, http.StatusBadRequest, "400 Bad Request", "Invalid user ID or target role.")
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO user_roles (user_id, role, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET role = excluded.role, updated_at = CURRENT_TIMESTAMP`,
		targetUserID, newRole,
	)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	reqStatus := "rejected"
	if newRole == "moderator" || newRole == "admin" {
		reqStatus = "approved"
	}
	_, _ = db.DB.Exec("UPDATE role_requests SET status = ? WHERE user_id = ?", reqStatus, targetUserID)

	http.Redirect(w, r, "/moderation?msg=User+role+updated", http.StatusSeeOther)
}

// ApprovePostHandler allows a moderator or administrator to approve a pending post so it becomes publicly visible.
func ApprovePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}
	session, err := CheckSession(r)
	if err != nil || !session.Loggedin || (session.Role != "moderator" && session.Role != "admin") {
		ErrorPageHandler(w, r, http.StatusForbidden, "403 Forbidden", "Moderator or administrator privileges are required.")
		return
	}

	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil || postID <= 0 {
		ErrorPageHandler(w, r, http.StatusBadRequest, "400 Bad Request", "Invalid post ID.")
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO post_moderation (post_id, status, reviewed_by, updated_at)
		VALUES (?, 'approved', ?, CURRENT_TIMESTAMP)
		ON CONFLICT(post_id) DO UPDATE SET status = 'approved', reviewed_by = excluded.reviewed_by, updated_at = CURRENT_TIMESTAMP`,
		postID, session.UserID,
	)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}
	http.Redirect(w, r, "/moderation?msg=Post+approved", http.StatusSeeOther)
}

// DeletePostModerationHandler allows a moderator or administrator to delete a post.
func DeletePostModerationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}
	session, err := CheckSession(r)
	if err != nil || !session.Loggedin || (session.Role != "moderator" && session.Role != "admin") {
		ErrorPageHandler(w, r, http.StatusForbidden, "403 Forbidden", "Moderator or administrator privileges are required to delete posts.")
		return
	}

	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil || postID <= 0 {
		ErrorPageHandler(w, r, http.StatusBadRequest, "400 Bad Request", "Invalid post ID.")
		return
	}

	if err := DeletePostByID(postID); err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}
	http.Redirect(w, r, "/moderation?msg=Post+deleted", http.StatusSeeOther)
}

// DeleteCommentModerationHandler allows an administrator (or moderator) to delete a comment.
func DeleteCommentModerationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}
	session, err := CheckSession(r)
	if err != nil || !session.Loggedin || (session.Role != "admin" && session.Role != "moderator") {
		ErrorPageHandler(w, r, http.StatusForbidden, "403 Forbidden", "Administrator or moderator privileges are required to delete comments.")
		return
	}

	commentID, err := strconv.Atoi(r.FormValue("comment_id"))
	if err != nil || commentID <= 0 {
		ErrorPageHandler(w, r, http.StatusBadRequest, "400 Bad Request", "Invalid comment ID.")
		return
	}

	_, _ = db.DB.Exec("DELETE FROM comment_reactions WHERE comment_id = ?", commentID)
	if _, err := db.DB.Exec("DELETE FROM comments WHERE id = ?", commentID); err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}
	http.Redirect(w, r, "/moderation?msg=Comment+deleted", http.StatusSeeOther)
}

// ReportPostHandler allows a moderator to report a post to the administrator with a reason category
// (irrelevant, obscene, illegal, or insulting).
func ReportPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}
	session, err := CheckSession(r)
	if err != nil || !session.Loggedin || (session.Role != "moderator" && session.Role != "admin") {
		ErrorPageHandler(w, r, http.StatusForbidden, "403 Forbidden", "Moderator privileges are required to submit reports to the administrator.")
		return
	}

	postID, err := strconv.Atoi(r.FormValue("post_id"))
	reason := strings.ToLower(strings.TrimSpace(r.FormValue("reason")))
	details := strings.TrimSpace(r.FormValue("details"))
	if err != nil || postID <= 0 || !validReportReasons[reason] {
		ErrorPageHandler(w, r, http.StatusBadRequest, "400 Bad Request", "Valid post ID and reason (irrelevant, obscene, illegal, or insulting) are required.")
		return
	}

	var exists int
	if err := db.DB.QueryRow("SELECT id FROM posts WHERE id = ?", postID).Scan(&exists); err == sql.ErrNoRows {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "Post not found.")
		return
	}

	if _, err := db.DB.Exec(
		"INSERT INTO moderation_reports (post_id, reporter_id, reason, details) VALUES (?, ?, ?, ?)",
		postID, session.UserID, reason, details,
	); err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}
	http.Redirect(w, r, "/moderation?msg=Report+sent+to+administrator", http.StatusSeeOther)
}

// RespondReportHandler allows an administrator to respond to a moderator's report (resolve or dismiss, and optionally delete the post).
func RespondReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}
	session, err := CheckSession(r)
	if err != nil || !session.Loggedin || session.Role != "admin" {
		ErrorPageHandler(w, r, http.StatusForbidden, "403 Forbidden", "Administrator privileges are required to respond to moderator reports.")
		return
	}

	reportID, err := strconv.Atoi(r.FormValue("report_id"))
	status := strings.TrimSpace(r.FormValue("status"))
	responseText := strings.TrimSpace(r.FormValue("admin_response"))
	deletePost := r.FormValue("delete_post") == "true"

	if err != nil || reportID <= 0 || (status != "resolved" && status != "dismissed") {
		ErrorPageHandler(w, r, http.StatusBadRequest, "400 Bad Request", "Invalid report ID or status.")
		return
	}

	var postID int
	if err := db.DB.QueryRow("SELECT post_id FROM moderation_reports WHERE id = ?", reportID).Scan(&postID); err != nil {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "Report not found.")
		return
	}

	if _, err := db.DB.Exec(
		"UPDATE moderation_reports SET status = ?, admin_response = ? WHERE id = ?",
		status, responseText, reportID,
	); err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	if deletePost && postID > 0 {
		_ = DeletePostByID(postID)
	}
	http.Redirect(w, r, "/moderation?msg=Report+updated", http.StatusSeeOther)
}

// ManageCategoriesHandler allows an administrator to create or delete forum categories.
func ManageCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}
	session, err := CheckSession(r)
	if err != nil || !session.Loggedin || session.Role != "admin" {
		ErrorPageHandler(w, r, http.StatusForbidden, "403 Forbidden", "Administrator privileges are required to manage categories.")
		return
	}

	action := strings.TrimSpace(r.FormValue("action"))
	switch action {
	case "create":
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			ErrorPageHandler(w, r, http.StatusBadRequest, "400 Bad Request", "Category name cannot be empty.")
			return
		}
		if _, err := db.DB.Exec("INSERT INTO categories (name) VALUES (?)", name); err != nil {
			ErrorPageHandler(w, r, http.StatusConflict, "409 Conflict", "Category already exists.")
			return
		}
	case "delete":
		catID, err := strconv.Atoi(r.FormValue("category_id"))
		if err != nil || catID <= 0 {
			ErrorPageHandler(w, r, http.StatusBadRequest, "400 Bad Request", "Invalid category ID.")
			return
		}
		_, _ = db.DB.Exec("DELETE FROM post_categories WHERE category_id = ?", catID)
		if _, err := db.DB.Exec("DELETE FROM categories WHERE id = ?", catID); err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			return
		}
	default:
		ErrorPageHandler(w, r, http.StatusBadRequest, "400 Bad Request", "Unknown category action.")
		return
	}

	http.Redirect(w, r, "/moderation?msg=Categories+updated", http.StatusSeeOther)
}
