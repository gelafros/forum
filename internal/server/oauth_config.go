package server

import (
	"log"
	"os"
	"strings"
)

// missingOAuthVars returns, per provider, the required OAuth environment variables that are not set.
func missingOAuthVars() map[string][]string {
	providers := []struct {
		name string
		vars []string
	}{
		{"GitHub", []string{"GITHUB_CLIENT_ID", "GITHUB_CLIENT_SECRET"}},
		{"Google", []string{"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET"}},
	}

	missing := make(map[string][]string)
	for _, p := range providers {
		for _, v := range p.vars {
			if strings.TrimSpace(os.Getenv(v)) == "" {
				missing[p.name] = append(missing[p.name], v)
			}
		}
	}
	return missing
}

// warnMissingOAuthConfig logs a startup warning for each OAuth provider that is not configured.
func warnMissingOAuthConfig() {
	missing := missingOAuthVars()
	for _, provider := range []string{"GitHub", "Google"} {
		if vars, ok := missing[provider]; ok {
			log.Printf("WARNING: %s OAuth not configured (missing %s) — %s login will be unavailable. See readme \"OAuth Setup\".",
				provider, strings.Join(vars, ", "), provider)
		}
	}
}
