package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoogleAndGitHubOAuthFlow(t *testing.T) {
	testDB := setupHandlerTestDB(t)

	// Spin up a mock OAuth2 provider server for both Google and GitHub.
	mockProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/google/token":
			fmt.Fprint(w, `{"access_token":"google-mock-token"}`)
		case "/google/userinfo":
			fmt.Fprint(w, `{"id":"g-12345","email":"googleuser@example.com","name":"Google User"}`)
		case "/github/token":
			fmt.Fprint(w, `{"access_token":"github-mock-token"}`)
		case "/github/user":
			fmt.Fprint(w, `{"id":98765,"login":"ghdeveloper","email":"ghdev@example.com"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockProvider.Close()

	prevGoogleToken, prevGoogleInfo := googleTokenURL, googleUserInfoURL
	prevGitHubToken, prevGitHubInfo := githubTokenURL, githubUserInfoURL
	googleTokenURL = mockProvider.URL + "/google/token"
	googleUserInfoURL = mockProvider.URL + "/google/userinfo"
	githubTokenURL = mockProvider.URL + "/github/token"
	githubUserInfoURL = mockProvider.URL + "/github/user"
	t.Cleanup(func() {
		googleTokenURL, googleUserInfoURL = prevGoogleToken, prevGoogleInfo
		githubTokenURL, githubUserInfoURL = prevGitHubToken, prevGitHubInfo
	})

	t.Setenv("GOOGLE_CLIENT_ID", "test-google-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-google-secret")
	t.Setenv("GITHUB_CLIENT_ID", "test-github-client")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-github-secret")

	// 1. Google Login redirect sets oauth_state cookie
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	GoogleLoginHandler(loginRec, loginReq)
	if loginRec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("GoogleLoginHandler status = %d, want %d", loginRec.Code, http.StatusTemporaryRedirect)
	}
	var stateCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == oauthStateCookieName {
			stateCookie = c
		}
	}
	if stateCookie == nil || stateCookie.Value == "" {
		t.Fatal("expected oauth_state cookie from GoogleLoginHandler")
	}

	// 2. Google Callback creates user, oauth_account, and session cookie
	cbReq := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test-code&state="+stateCookie.Value, nil)
	cbReq.AddCookie(stateCookie)
	cbRec := httptest.NewRecorder()
	GoogleCallbackHandler(cbRec, cbReq)
	if cbRec.Code != http.StatusSeeOther {
		t.Fatalf("GoogleCallbackHandler status = %d, want %d", cbRec.Code, http.StatusSeeOther)
	}

	var count int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM oauth_accounts WHERE provider = 'google' AND provider_user_id = 'g-12345'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("expected google oauth_account row, got count=%d err=%v", count, err)
	}

	// 3. GitHub Login + Callback creates user, oauth_account, and session cookie
	ghLoginReq := httptest.NewRequest(http.MethodGet, "/auth/github/login", nil)
	ghLoginRec := httptest.NewRecorder()
	GitHubLoginHandler(ghLoginRec, ghLoginReq)
	if ghLoginRec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("GitHubLoginHandler status = %d, want %d", ghLoginRec.Code, http.StatusTemporaryRedirect)
	}
	var ghState *http.Cookie
	for _, c := range ghLoginRec.Result().Cookies() {
		if c.Name == oauthStateCookieName {
			ghState = c
		}
	}
	ghCbReq := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=gh-code&state="+ghState.Value, nil)
	ghCbReq.AddCookie(ghState)
	ghCbRec := httptest.NewRecorder()
	GitHubCallbackHandler(ghCbRec, ghCbReq)
	if ghCbRec.Code != http.StatusSeeOther {
		t.Fatalf("GitHubCallbackHandler status = %d, want %d", ghCbRec.Code, http.StatusSeeOther)
	}

	var ghUsername string
	if err := testDB.QueryRow("SELECT username FROM users WHERE email = 'ghdev@example.com'").Scan(&ghUsername); err != nil || !strings.HasPrefix(ghUsername, "ghdeveloper") {
		t.Fatalf("expected github user ghdeveloper, got %q (err=%v)", ghUsername, err)
	}
}
