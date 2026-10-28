CREATE TABLE IF NOT EXISTS post_images (
    id INTEGER PRIMARY KEY,
    post_id INTEGER NOT NULL UNIQUE,
    image_path TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES posts(id)
);
