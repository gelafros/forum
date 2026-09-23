CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY, -- post_id
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, -- adding Creation Time from SQL
    FOREIGN KEY (user_id) REFERENCES users(id) -- the value of user_id field takes the value from reference ID of "users" TABLE (users(id))
);