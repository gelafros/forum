package handlers

import (
	"database/sql"
	"forum/internal/db"
	"net/http"
)

type Session struct {
	UserID   int
	Username string
	Loggedin bool
}

// Check cookie to determine logged-in state
func CheckSession(r *http.Request) (Session, error) {

	var (
		userid   int
		username string
		session  Session
	)

	cookie, err := r.Cookie("session_token")
	if err == nil {

		// Cookie present — validate the session against the database
		token := cookie.Value
		qErr := db.DB.QueryRow(`
			SELECT users.id, users.username
			FROM sessions
			JOIN users ON sessions.user_id = users.id
			WHERE sessions.token = ? AND sessions.expires_at > CURRENT_TIMESTAMP`, token).Scan(&userid, &username)
		if qErr == nil {

			session = Session{
				UserID:   userid,
				Username: username,
				Loggedin: true,
			}
		} else if qErr != sql.ErrNoRows {

			session = Session{
				UserID:   0,
				Username: "",
				Loggedin: false,
			}
			return session, qErr
		}
	}
	return session, nil
}
