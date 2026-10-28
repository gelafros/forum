package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
