package handlers

import (
	"database/sql"
	"forum/internal/db"
	"log"
	"net/http"
	"strconv"
)

// Handles Like / Dislike reaction for a post
func PostReactHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		log.Printf("PostReactHandler: method %s not allowed\n", r.Method)
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
		ErrorPageHandler(w, r, http.StatusUnauthorized, "401 Unauthorized", "You must be logged in to react to a post.")
		return
	}

	// Read PostID
	formPostID := r.FormValue("post_id")
	if formPostID == "" {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Bad Request", "Post ID is required.")
		return
	}
	postID, err := strconv.Atoi(formPostID)
	if err != nil || postID <= 0 {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Bad Request", "Invalid post ID.")
		log.Println("Error converting postID:", err)
		return
	}

	// Check PostID
	var postIDScn int
	err = db.DB.QueryRow("SELECT posts.id FROM posts WHERE posts.id = ?", postID).Scan(&postIDScn)

	if err == sql.ErrNoRows {
		ErrorPageHandler(w, r, http.StatusNotFound, "Page Not Found", "Post not found.")
		return
	}

	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("PostReactHandler query error:", err)
		return
	}

	// Check reaction
	formReaction := r.FormValue("reaction_type")
	if formReaction != "like" && formReaction != "dislike" {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Bad Request", "Bad request on post reactions")
		return
	}

	var scannedReaction string
	// Hold the "reaction_type" from post_reactions table
	err = db.DB.QueryRow("SELECT reaction_type FROM post_reactions pr WHERE pr.user_id = ? AND pr.post_id = ?", session.UserID, postID).Scan(&scannedReaction)

	// 1. Empty post reaction case
	if err == sql.ErrNoRows {
		_, err = db.DB.Exec("INSERT INTO post_reactions (user_id, post_id, reaction_type) VALUES (?, ?, ?)", session.UserID, postID, formReaction)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("PostReactHandler insert reaction in post error:", err)
			return
		}
		http.Redirect(w, r, "/post?id="+strconv.Itoa(postID), http.StatusSeeOther)
		return
	}

	// 2. Other errors handling
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("PostReactHandler query error:", err)
		return
	}

	// 3. NOT empty post reaction case (change reaction)

	// Delete reaction if it is the same as before
	if scannedReaction == formReaction {
		_, err = db.DB.Exec("DELETE FROM post_reactions WHERE user_id = ? AND post_id = ?", session.UserID, postID)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("PostReactHandler delete reaction in post error:", err)
			return
		}
		http.Redirect(w, r, "/post?id="+strconv.Itoa(postID), http.StatusSeeOther)
		return
	} else {
		// Update reaction if it is differently from before
		_, err = db.DB.Exec("UPDATE post_reactions SET reaction_type = ? WHERE user_id = ? AND post_id = ?", formReaction, session.UserID, postID)
		if err != nil {
			ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
			log.Println("PostReactHandler update reaction in post error:", err)
			return
		}
		http.Redirect(w, r, "/post?id="+strconv.Itoa(postID), http.StatusSeeOther)
		return
	}

}
