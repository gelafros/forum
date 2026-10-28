package handlers

import (
	"database/sql"
	"log"
	"net/http"

	"forum/internal/db"
)

type Session struct {
	UserID   int
	Username string
	Role     string
	Loggedin bool
}

// CheckSession inspects the session_token cookie and returns the user's Session and Role.
func CheckSession(r *http.Request) (Session, error) {
	cookie, err := r.Cookie("session_token")
	if err == http.ErrNoCookie {
		return Session{Role: "guest", Loggedin: false}, nil
	}
	if err != nil {
		log.Println("Cookie check error:", err)
		return Session{Role: "guest", Loggedin: false}, err
	}

	token := cookie.Value
	var (
		userid   int
		username string
		role     string
	)
	err = db.DB.QueryRow(`
		SELECT users.id, users.username, COALESCE(ur.role, 'user')
		FROM sessions
		JOIN users ON sessions.user_id = users.id
		LEFT JOIN user_roles ur ON ur.user_id = users.id
		WHERE sessions.token = ? AND datetime(sessions.expires_at) > CURRENT_TIMESTAMP`,
		token,
	).Scan(&userid, &username, &role)

	if err == sql.ErrNoRows {
		return Session{Role: "guest", Loggedin: false}, nil
	}
	if err != nil {
		log.Println("CheckSession QueryRow error:", err)
		return Session{Role: "guest", Loggedin: false}, err
	}

	return Session{
		UserID:   userid,
		Username: username,
		Role:     role,
		Loggedin: true,
	}, nil
}
