CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY, -- category_id
    name TEXT NOT NULL UNIQUE
);