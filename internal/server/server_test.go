package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMissingOAuthVars(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "id")
	t.Setenv("GITHUB_CLIENT_SECRET", "secret")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")

	missing := missingOAuthVars()
	if _, ok := missing["GitHub"]; ok {
		t.Errorf("GitHub reported missing although configured: %v", missing["GitHub"])
	}
	if got := len(missing["Google"]); got != 2 {
		t.Errorf("Google missing vars = %d, want 2", got)
	}
}

func TestLoadDotEnv(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	content := "# comment\nGITHUB_CLIENT_ID=\"test-gh-id\"\nexport GOOGLE_CLIENT_ID='test-google-id'\n"
	if err := os.WriteFile(envFile, []byte(content), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	_ = os.Unsetenv("GITHUB_CLIENT_ID")
	_ = os.Unsetenv("GOOGLE_CLIENT_ID")
	t.Cleanup(func() {
		_ = os.Unsetenv("GITHUB_CLIENT_ID")
		_ = os.Unsetenv("GOOGLE_CLIENT_ID")
	})

	if err := LoadDotEnv(envFile); err != nil {
		t.Fatalf("LoadDotEnv error: %v", err)
	}
	if got := os.Getenv("GITHUB_CLIENT_ID"); got != "test-gh-id" {
		t.Errorf("GITHUB_CLIENT_ID = %q, want %q", got, "test-gh-id")
	}
	if got := os.Getenv("GOOGLE_CLIENT_ID"); got != "test-google-id" {
		t.Errorf("GOOGLE_CLIENT_ID = %q, want %q", got, "test-google-id")
	}
}

func TestRegisterRoutesUsesDefaultMux(t *testing.T) {
	// Verifies that every application URL is registered on the default server mux.
	previousMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() { http.DefaultServeMux = previousMux })
	registerRoutes()

	cases := []struct {
		path        string
		wantPattern string
	}{
		{"/", "/"},
		{"/register", "/register"},
		{"/login", "/login"},
		{"/logout", "/logout"},
		{"/auth/google/login", "/auth/google/login"},
		{"/auth/google/callback", "/auth/google/callback"},
		{"/auth/github/login", "/auth/github/login"},
		{"/auth/github/callback", "/auth/github/callback"},
		{"/create_post", "/create_post"},
		{"/post", "/post"},
		{"/post/react", "/post/react"},
		{"/comment/react", "/comment/react"},
		{"/uploads/sample.png", "/uploads/"},
		{"/moderation", "/moderation"},
		{"/moderation/request-role", "/moderation/request-role"},
		{"/moderation/role", "/moderation/role"},
		{"/moderation/post/approve", "/moderation/post/approve"},
		{"/moderation/post/delete", "/moderation/post/delete"},
		{"/moderation/comment/delete", "/moderation/comment/delete"},
		{"/moderation/report", "/moderation/report"},
		{"/moderation/report/respond", "/moderation/report/respond"},
		{"/moderation/categories", "/moderation/categories"},
		{"/templates/style.css", "/templates/style.css"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			_, pattern := http.DefaultServeMux.Handler(request)
			if pattern != tc.wantPattern {
				t.Errorf("pattern for %q = %q, want %q", tc.path, pattern, tc.wantPattern)
			}
		})
	}
}
