-- Cookies Creation

CREATE TABLE sessions (
    id INTEGER PRIMARY KEY, -- sessions_id
    user_id INTEGER NOT NULL,
    token TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (user_id) -- Prevent more than one session record per user.
);