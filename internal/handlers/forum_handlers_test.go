package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func addSession(t *testing.T, testDB *sql.DB, userID int, token string) {
	t.Helper()
	if _, err := testDB.Exec("INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)", userID, token, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func addPost(t *testing.T, testDB *sql.DB, userID int, title string) int {
	t.Helper()
	result, err := testDB.Exec("INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)", userID, title, "content for "+title)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return int(id)
}

func TestRegisterHandlerRejectsDuplicateEmail(t *testing.T) {
	// Verifies that a duplicate email returns 409 and does not create another user.
	testDB := setupHandlerTestDB(t)
	addUser(t, testDB, "alex", "alex@example.com", "secret")

	form := url.Values{"username": {"other"}, "email": {"alex@example.com"}, "password": {"secret"}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	RegisterHandler(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	var count int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("user count = %d, want 1", count)
	}
}

func TestCreatePostAndComment(t *testing.T) {
	// Covers authenticated post creation, category links, and comment creation.
	testDB := setupHandlerTestDB(t)
	userID := addUser(t, testDB, "alex", "alex@example.com", "secret")
	addSession(t, testDB, userID, "active-token")

	form := url.Values{"title": {"My post"}, "content": {"Hello forum"}, "categories": {"1", "2"}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/create_post", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: "session_token", Value: "active-token"})
	CreatePostHandler(recorder, request)
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/" {
		t.Errorf("post response = %d, %q; want 303, /", recorder.Code, recorder.Header().Get("Location"))
	}

	var postID, categories int
	if err := testDB.QueryRow("SELECT id FROM posts WHERE title = ?", "My post").Scan(&postID); err != nil {
		t.Fatal(err)
	}
	if err := testDB.QueryRow("SELECT COUNT(*) FROM post_categories WHERE post_id = ?", postID).Scan(&categories); err != nil {
		t.Fatal(err)
	}
	if categories != 2 {
		t.Errorf("category links = %d, want 2", categories)
	}

	commentForm := url.Values{"content": {"A useful comment"}}
	commentRecorder := httptest.NewRecorder()
	commentRequest := httptest.NewRequest(http.MethodPost, "/post?id="+strconv.Itoa(postID), strings.NewReader(commentForm.Encode()))
	commentRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	commentRequest.AddCookie(&http.Cookie{Name: "session_token", Value: "active-token"})
	PostDetailHandler(commentRecorder, commentRequest)
	if commentRecorder.Code != http.StatusSeeOther || commentRecorder.Header().Get("Location") != "/post?id="+strconv.Itoa(postID) {
		t.Errorf("comment response = %d, %q", commentRecorder.Code, commentRecorder.Header().Get("Location"))
	}
	var commentCount int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM comments WHERE post_id = ? AND content = ?", postID, "A useful comment").Scan(&commentCount); err != nil {
		t.Fatal(err)
	}
	if commentCount != 1 {
		t.Errorf("comment count = %d, want 1", commentCount)
	}
}

func TestReactionSwitching(t *testing.T) {
	// Verifies the shared reaction behavior: insert like, toggle it off, then switch dislike to like.
	testDB := setupHandlerTestDB(t)
	userID := addUser(t, testDB, "alex", "alex@example.com", "secret")
	addSession(t, testDB, userID, "active-token")
	postID := addPost(t, testDB, userID, "Post")
	result, err := testDB.Exec("INSERT INTO comments (user_id, post_id, content) VALUES (?, ?, ?)", userID, postID, "Comment")
	if err != nil {
		t.Fatal(err)
	}
	commentID64, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		handler   http.HandlerFunc
		path      string
		idName    string
		id        int
		table     string
		column    string
		reactions []string
		want      string
	}{
		{"post", PostReactHandler, "/post/react", "post_id", postID, "post_reactions", "post_id", []string{"like", "like", "dislike", "like"}, "like"},
		{"comment", CommentReactHandler, "/comment/react", "comment_id", int(commentID64), "comment_reactions", "comment_id", []string{"like", "like", "dislike", "like"}, "like"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, reaction := range tc.reactions {
				form := url.Values{tc.idName: {strconv.Itoa(tc.id)}, "reaction_type": {reaction}}
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(form.Encode()))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				request.AddCookie(&http.Cookie{Name: "session_token", Value: "active-token"})
				tc.handler(recorder, request)
				if recorder.Code != http.StatusSeeOther {
					t.Fatalf("reaction %q status = %d, want 303", reaction, recorder.Code)
				}
			}
			var stored string
			if err := testDB.QueryRow("SELECT reaction_type FROM "+tc.table+" WHERE user_id = ? AND "+tc.column+" = ?", userID, tc.id).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if stored != tc.want {
				t.Errorf("stored reaction = %q, want %q", stored, tc.want)
			}
		})
	}
}

func TestLoadPostsFilters(t *testing.T) {
	// Covers category, created-by-me, and liked-by-me filters without real forum data.
	testDB := setupHandlerTestDB(t)
	alexID := addUser(t, testDB, "alex", "alex@example.com", "secret")
	beaID := addUser(t, testDB, "bea", "bea@example.com", "secret")
	alexPost := addPost(t, testDB, alexID, "Alex post")
	beaPost := addPost(t, testDB, beaID, "Bea post")
	if _, err := testDB.Exec("INSERT INTO post_categories (post_id, category_id) VALUES (?, 1)", alexPost); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Exec("INSERT INTO post_reactions (user_id, post_id, reaction_type) VALUES (?, ?, 'like')", alexID, beaPost); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		categoryID int
		filter     string
		wantID     int
	}{
		{"category", 1, "", alexPost},
		{"created", 0, "created", alexPost},
		{"liked", 0, "liked", beaPost},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			posts, err := loadPosts(tc.categoryID, tc.filter, alexID)
			if err != nil {
				t.Fatal(err)
			}
			if len(posts) != 1 || posts[0].ID != tc.wantID {
				t.Errorf("posts = %+v, want only post %d", posts, tc.wantID)
			}
		})
	}
}
