package handlers

import (
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
		ErrorPageHandler(w, r, http.StatusNotFound, "Page Not Found", "Null post")
		return
	}
	postID, err := strconv.Atoi(formPostID)
	if err != nil || postID <= 0 {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		log.Println("Error converting postID:", err)
		return
	}

	// Check PostID

	// Check reaction
	reaction := r.FormValue("reaction_type")
	if reaction != "like" || reaction == "dislike" {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Bad Request", "Bad request on post reactions")
		return
	}

}
