CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY, -- user_id
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL
);