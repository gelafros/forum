package handlers

import (
	"database/sql"
	"forum/internal/db"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Handles Like / Dislike reaction for a Comment
func CommentReactHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		log.Printf("CommentRectHandler: method %s not allowed\n", r.Method)
		return
	}

	// Determine logged-in state from session cookie
	session, err := CheckSession(r)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	// Unauthorized user Error 401 (guest)
	if !session.Loggedin {
		ErrorPageHandler(w, r, http.StatusUnauthorized, "401 Unauthorized", "You must be logged in to react to a comment.")
		return
	}

	// Read CommentID
	formCommentID := r.FormValue("comment_id")
	if formCommentID == "" {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "Comment ID is required.")
		return
	}
	commentID, err := strconv.Atoi(formCommentID)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "Comment ID is required.")
		log.Println("Error converting commentID:", err)
		return
	}

	// Check Comment
	var postIDScn int

	err = db.DB.QueryRow("SELECT post_id FROM comments WHERE comments.id = ?", commentID).Scan(&postIDScn) // SELECT the post.ID from formCommentID if it exists
	if err == sql.ErrNoRows {
		ErrorPageHandler(w, r, http.StatusNotFound, "Page Not Found", "Comment not found.")
		return
	}

	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("CommentReactHandler query error:", err)
		return
	}

	// Check reaction
	formReaction := r.FormValue("reaction_type")
	if formReaction != "like" && formReaction != "dislike" {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Bad Request", "Bad request on comment reactions")
		return
	}

	// Determine safe redirect destination
	redirectURL := r.FormValue("redirect_url") // this return "" because there isn't "redirect_url" in post_detail.html
	if redirectURL == "" || !strings.HasPrefix(redirectURL, "/") || strings.HasPrefix(redirectURL, "//") {
		redirectURL = "/post?id=" + strconv.Itoa(postIDScn)
	}

	// Reaction procedure
	var scannedReaction string

	err = db.DB.QueryRow("SELECT reaction_type FROM comment_reactions cr WHERE cr.user_id = ? AND cr.comment_id = ?", session.UserID, commentID).Scan(&scannedReaction) // Session.UserID is the current online user

	// 1. Empty comment reaction case
	if err == sql.ErrNoRows {
		_, err = db.DB.Exec("INSERT INTO comment_reactions (user_id, comment_id, reaction_type) VALUES (?, ?, ?)", session.UserID, commentID, formReaction)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("CommentReactHandler insert reaction in comment error:", err)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	// 2. Other errors handling
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("CommentReactHandler query error:", err)
		return
	}

	// 3. NOT empty comment reaction case (change reaction)

	// Delete reaction if it is the same as before
	if scannedReaction == formReaction {
		_, err = db.DB.Exec("DELETE FROM comment_reactions WHERE user_id = ? AND comment_id = ?", session.UserID, commentID)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("CommentReactHandler delete reaction in comment error:", err)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	} else {
		// Update reaction if it is differently from before
		_, err = db.DB.Exec("UPDATE comment_reactions SET reaction_type = ? WHERE user_id = ? AND comment_id = ?", formReaction, session.UserID, commentID)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("CommentReactHandler update reaction in comment error:", err)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}
}
