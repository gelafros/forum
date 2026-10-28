package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"forum/internal/db"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Configurable OAuth2 provider URLs (overridable in unit tests).
var (
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

	githubAuthURL     = "https://github.com/login/oauth/authorize"
	githubTokenURL    = "https://github.com/login/oauth/access_token"
	githubUserInfoURL = "https://api.github.com/user"
	githubEmailsURL   = "https://api.github.com/user/emails"

	oauthHTTPClient = &http.Client{Timeout: 10 * time.Second}
)

const oauthStateCookieName = "oauth_state"

type oauthUserInfo struct {
	Provider       string
	ProviderUserID string
	Email          string
	Username       string
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// issueOAuthState creates a random state token and stores it in an HttpOnly cookie to prevent CSRF.
func issueOAuthState(w http.ResponseWriter) (string, error) {
	rawState, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	state := rawState.String()
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("COOKIE_SECURE") == "true",
	})
	return state, nil
}

// verifyOAuthState validates the callback state parameter against the stored state cookie and clears it.
func verifyOAuthState(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return r.URL.Query().Get("state") == cookie.Value
}

// GoogleLoginHandler initiates the Google OAuth 2.0 authorization flow.
func GoogleLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		ErrorPageHandler(w, r, http.StatusServiceUnavailable, "OAuth Not Configured", "Google authentication is not configured on this server (missing GOOGLE_CLIENT_ID).")
		return
	}

	state, err := issueOAuthState(w)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	redirectURI := envOrDefault("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback")
	params := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"access_type":   {"online"},
	}
	http.Redirect(w, r, googleAuthURL+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

// GoogleCallbackHandler handles the Google OAuth 2.0 callback, exchanges the authorization code,
// fetches the Google profile, links or registers the user, and starts a forum session.
func GoogleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}

	if !verifyOAuthState(w, r) {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Invalid OAuth State", "The authentication request expired or had an invalid state token.")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Authentication Cancelled", "No authorization code was received from Google.")
		return
	}

	tokenForm := url.Values{
		"code":          {code},
		"client_id":     {os.Getenv("GOOGLE_CLIENT_ID")},
		"client_secret": {os.Getenv("GOOGLE_CLIENT_SECRET")},
		"redirect_uri":  {envOrDefault("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback")},
		"grant_type":    {"authorization_code"},
	}

	accessToken, err := exchangeOAuthCode(googleTokenURL, tokenForm)
	if err != nil {
		log.Println("GoogleCallbackHandler token exchange error:", err)
		ErrorPageHandler(w, r, http.StatusBadGateway, "Authentication Failed", "Failed to verify authorization code with Google.")
		return
	}

	info, err := fetchGoogleUser(accessToken)
	if err != nil {
		log.Println("GoogleCallbackHandler userinfo error:", err)
		ErrorPageHandler(w, r, http.StatusBadGateway, "Authentication Failed", "Failed to retrieve profile information from Google.")
		return
	}

	completeOAuthLogin(w, r, info)
}

// GitHubLoginHandler initiates the GitHub OAuth 2.0 authorization flow.
func GitHubLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}

	clientID := os.Getenv("GITHUB_CLIENT_ID")
	if clientID == "" {
		ErrorPageHandler(w, r, http.StatusServiceUnavailable, "OAuth Not Configured", "GitHub authentication is not configured on this server (missing GITHUB_CLIENT_ID).")
		return
	}

	state, err := issueOAuthState(w)
	if err != nil {
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	redirectURI := envOrDefault("GITHUB_REDIRECT_URL", "http://localhost:8080/auth/github/callback")
	params := url.Values{
		"client_id":    {clientID},
		"redirect_uri": {redirectURI},
		"scope":        {"read:user user:email"},
		"state":        {state},
	}
	http.Redirect(w, r, githubAuthURL+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

// GitHubCallbackHandler handles the GitHub OAuth 2.0 callback, exchanges the authorization code,
// fetches the GitHub user profile, links or registers the user, and starts a forum session.
func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}

	if !verifyOAuthState(w, r) {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Invalid OAuth State", "The authentication request expired or had an invalid state token.")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		ErrorPageHandler(w, r, http.StatusBadRequest, "Authentication Cancelled", "No authorization code was received from GitHub.")
		return
	}

	tokenForm := url.Values{
		"code":          {code},
		"client_id":     {os.Getenv("GITHUB_CLIENT_ID")},
		"client_secret": {os.Getenv("GITHUB_CLIENT_SECRET")},
		"redirect_uri":  {envOrDefault("GITHUB_REDIRECT_URL", "http://localhost:8080/auth/github/callback")},
	}

	accessToken, err := exchangeOAuthCode(githubTokenURL, tokenForm)
	if err != nil {
		log.Println("GitHubCallbackHandler token exchange error:", err)
		ErrorPageHandler(w, r, http.StatusBadGateway, "Authentication Failed", "Failed to verify authorization code with GitHub.")
		return
	}

	info, err := fetchGitHubUser(accessToken)
	if err != nil {
		log.Println("GitHubCallbackHandler userinfo error:", err)
		ErrorPageHandler(w, r, http.StatusBadGateway, "Authentication Failed", "Failed to retrieve profile information from GitHub.")
		return
	}

	completeOAuthLogin(w, r, info)
}

func exchangeOAuthCode(tokenURL string, form url.Values) (string, error) {
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", errors.New("empty access_token in response")
	}
	return payload.AccessToken, nil
}

func fetchGoogleUser(accessToken string) (oauthUserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return oauthUserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return oauthUserInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oauthUserInfo{}, fmt.Errorf("google userinfo status %d", resp.StatusCode)
	}

	var profile struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&profile); err != nil {
		return oauthUserInfo{}, err
	}
	if profile.ID == "" || profile.Email == "" {
		return oauthUserInfo{}, errors.New("incomplete google profile")
	}

	username := strings.TrimSpace(profile.Name)
	if username == "" {
		username = strings.Split(profile.Email, "@")[0]
	}

	return oauthUserInfo{
		Provider:       "google",
		ProviderUserID: profile.ID,
		Email:          profile.Email,
		Username:       username,
	}, nil
}

func fetchGitHubUser(accessToken string) (oauthUserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, githubUserInfoURL, nil)
	if err != nil {
		return oauthUserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return oauthUserInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oauthUserInfo{}, fmt.Errorf("github user status %d", resp.StatusCode)
	}

	var profile struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&profile); err != nil {
		return oauthUserInfo{}, err
	}
	if profile.ID == 0 || profile.Login == "" {
		return oauthUserInfo{}, errors.New("incomplete github profile")
	}

	email := strings.TrimSpace(profile.Email)
	if email == "" {
		email, _ = fetchGitHubPrimaryEmail(accessToken)
	}
	if email == "" {
		email = fmt.Sprintf("%s@users.noreply.github.com", profile.Login)
	}

	return oauthUserInfo{
		Provider:       "github",
		ProviderUserID: strconv.FormatInt(profile.ID, 10),
		Email:          email,
		Username:       profile.Login,
	}, nil
}

func fetchGitHubPrimaryEmail(accessToken string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, githubEmailsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails status %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary && e.Verified && e.Email != "" {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified && e.Email != "" {
			return e.Email, nil
		}
	}
	return "", errors.New("no verified github email")
}

func findOrCreateOAuthUser(info oauthUserInfo) (int, string, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback()

	var userID int
	var username string
	err = tx.QueryRow(`
		SELECT u.id, u.username
		FROM oauth_accounts oa
		JOIN users u ON u.id = oa.user_id
		WHERE oa.provider = ? AND oa.provider_user_id = ?`,
		info.Provider, info.ProviderUserID,
	).Scan(&userID, &username)
	if err == nil {
		return userID, username, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, "", err
	}

	// Check if a user with the same email already exists and link their account.
	err = tx.QueryRow("SELECT id, username FROM users WHERE email = ?", info.Email).Scan(&userID, &username)
	if errors.Is(err, sql.ErrNoRows) {
		baseName := sanitizeUsername(info.Username)
		uniqueName, uErr := resolveUniqueUsername(tx, baseName)
		if uErr != nil {
			return 0, "", uErr
		}

		randomSecret, sErr := uuid.NewRandom()
		if sErr != nil {
			return 0, "", sErr
		}
		hash, hErr := bcrypt.GenerateFromPassword([]byte(randomSecret.String()), bcrypt.DefaultCost)
		if hErr != nil {
			return 0, "", hErr
		}

		res, iErr := tx.Exec(
			"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
			uniqueName, info.Email, string(hash),
		)
		if iErr != nil {
			return 0, "", iErr
		}
		id64, lErr := res.LastInsertId()
		if lErr != nil {
			return 0, "", lErr
		}
		userID = int(id64)
		username = uniqueName
	} else if err != nil {
		return 0, "", err
	}

	if _, err := tx.Exec(
		"INSERT INTO oauth_accounts (user_id, provider, provider_user_id) VALUES (?, ?, ?)",
		userID, info.Provider, info.ProviderUserID,
	); err != nil {
		return 0, "", err
	}

	if err := tx.Commit(); err != nil {
		return 0, "", err
	}
	return userID, username, nil
}

func sanitizeUsername(raw string) string {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.ReplaceAll(cleaned, " ", "_")
	if cleaned == "" {
		return "user"
	}
	return cleaned
}

func resolveUniqueUsername(tx *sql.Tx, base string) (string, error) {
	candidate := base
	for i := 1; i <= 100; i++ {
		var count int
		if err := tx.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", candidate).Scan(&count); err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s_%d", base, i)
	}
	suffix, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return base + "_" + suffix.String()[:8], nil
}

func completeOAuthLogin(w http.ResponseWriter, r *http.Request, info oauthUserInfo) {
	userID, username, err := findOrCreateOAuthUser(info)
	if err != nil {
		log.Println("completeOAuthLogin user error:", err)
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	if _, err := db.DB.Exec("DELETE FROM sessions WHERE user_id = ?", userID); err != nil {
		log.Println("completeOAuthLogin delete old session error:", err)
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	rawToken, err := uuid.NewRandom()
	if err != nil {
		log.Println("completeOAuthLogin uuid error:", err)
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}
	token := rawToken.String()
	expiresAt := time.Now().Add(24 * time.Hour)

	if _, err := db.DB.Exec(
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, token, expiresAt,
	); err != nil {
		log.Println("completeOAuthLogin insert session error:", err)
		ErrorPageHandler(w, r, http.StatusInternalServerError, "Internal Server Error", errInternal)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("COOKIE_SECURE") == "true",
	})

	log.Printf("The user %q has logged in via %s OAuth...", username, info.Provider)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
