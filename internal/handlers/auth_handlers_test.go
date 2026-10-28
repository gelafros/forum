package handlers

import (
	"database/sql"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"forum/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

func setupHandlerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	t.Chdir(filepath.Join("..", ".."))

	testDB, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "forum-test.db")+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	testDB.SetMaxOpenConns(1)

	files, err := os.ReadDir("internal/db/migrations")
	if err != nil {
		testDB.Close()
		t.Fatal(err)
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join("internal/db/migrations", file.Name()))
		if err != nil {
			testDB.Close()
			t.Fatal(err)
		}
		if _, err := testDB.Exec(string(sqlBytes)); err != nil {
			testDB.Close()
			t.Fatalf("apply %s: %v", file.Name(), err)
		}
	}

	previousDB := db.DB
	db.DB = testDB
	t.Cleanup(func() {
		db.DB = previousDB
		testDB.Close()
	})
	return testDB
}

func addUser(t *testing.T, testDB *sql.DB, username, email, password string) int {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	result, err := testDB.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)", username, email, string(hash))
	if err != nil {
		t.Fatal(err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return int(userID)
}

func TestHandlersRejectUnsupportedMethods(t *testing.T) {
	// Verifies that each handler rejects unsupported HTTP methods with 405
	// and announces its accepted methods through the Allow response header.
	cases := []struct {
		name    string
		handler http.HandlerFunc
		path    string
		allow   string
	}{
		{"home", HomeHandler, "/", "GET"},
		{"register", RegisterHandler, "/register", "GET, POST"},
		{"login", LoginHandler, "/login", "GET, POST"},
		{"create post", CreatePostHandler, "/create_post", "GET, POST"},
		{"post reaction", PostReactHandler, "/post/react", "POST"},
		{"comment reaction", CommentReactHandler, "/comment/react", "POST"},
		{"logout", LogoutHandler, "/logout", "POST"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(filepath.Join("..", ".."))
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, tc.path, nil)

			tc.handler(recorder, request)

			if recorder.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
			}
			if allow := recorder.Header().Get("Allow"); allow != tc.allow {
				t.Errorf("Allow = %q, want %q", allow, tc.allow)
			}
		})
	}
}

func TestCheckSession(t *testing.T) {
	// Covers guest requests, valid session tokens, and expired tokens.
	// An expired token must be treated as a guest session.
	testDB := setupHandlerTestDB(t)
	userID := addUser(t, testDB, "alex", "alex@example.com", "secret")

	if _, err := testDB.Exec("INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)", userID, "valid-token", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)", "expired", "expired@example.com", "secret"); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Exec("INSERT INTO sessions (user_id, token, expires_at) VALUES ((SELECT id FROM users WHERE username = ?), ?, ?)", "expired", "expired-token", time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		cookie     string
		wantUserID int
		wantName   string
		wantLogged bool
	}{
		{"no cookie", "", 0, "", false},
		{"valid token", "valid-token", userID, "alex", true},
		{"expired token", "expired-token", 0, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.cookie != "" {
				request.AddCookie(&http.Cookie{Name: "session_token", Value: tc.cookie})
			}

			session, err := CheckSession(request)
			if err != nil {
				t.Fatal(err)
			}
			if session.UserID != tc.wantUserID || session.Username != tc.wantName || session.Loggedin != tc.wantLogged {
				t.Errorf("session = %+v, want userID=%d username=%q loggedin=%t", session, tc.wantUserID, tc.wantName, tc.wantLogged)
			}
		})
	}
}

func TestLoginHandler(t *testing.T) {
	// Confirms successful login replaces an older session and sends a secure cookie.
	// It also confirms invalid credentials create neither a session nor a cookie.
	t.Run("success creates one session and cookie", func(t *testing.T) {
		testDB := setupHandlerTestDB(t)
		userID := addUser(t, testDB, "alex", "alex@example.com", "secret")
		if _, err := testDB.Exec("INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)", userID, "old-token", time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}

		form := url.Values{"identifier": {"alex"}, "password": {"secret"}}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		LoginHandler(recorder, request)

		if recorder.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
		}
		if location := recorder.Header().Get("Location"); location != "/" {
			t.Errorf("Location = %q, want /", location)
		}
		cookies := recorder.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != "session_token" || cookies[0].Value == "" || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
			t.Errorf("unexpected session cookie: %+v", cookies)
		}

		var count int
		if err := testDB.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = ?", userID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("session count = %d, want 1", count)
		}
	})

	t.Run("invalid credentials create no session", func(t *testing.T) {
		testDB := setupHandlerTestDB(t)
		userID := addUser(t, testDB, "alex", "alex@example.com", "secret")

		form := url.Values{"identifier": {"alex"}, "password": {"wrong"}}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		LoginHandler(recorder, request)

		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if cookies := recorder.Result().Cookies(); len(cookies) != 0 {
			t.Errorf("cookies = %+v, want none", cookies)
		}
		var count int
		if err := testDB.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = ?", userID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("session count = %d, want 0", count)
		}
	})
}

func TestLogoutHandlerDeletesSessionAndCookie(t *testing.T) {
	// Verifies that logout removes the stored session, clears the browser cookie,
	// and uses a 303 redirect back to the homepage.
	testDB := setupHandlerTestDB(t)
	userID := addUser(t, testDB, "alex", "alex@example.com", "secret")
	if _, err := testDB.Exec("INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)", userID, "active-token", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.AddCookie(&http.Cookie{Name: "session_token", Value: "active-token"})

	LogoutHandler(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if location := recorder.Header().Get("Location"); location != "/" {
		t.Errorf("Location = %q, want /", location)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session_token" || cookies[0].MaxAge != -1 || !cookies[0].Expires.Before(time.Now()) {
		t.Errorf("unexpected cleared cookie: %+v", cookies)
	}
	var count int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = ?", userID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("session count = %d, want 0", count)
	}
}
