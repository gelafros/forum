CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY, -- category_id
    name TEXT NOT NULL UNIQUE
);

-- Inserting if not exist category Names
INSERT OR IGNORE INTO categories (name) VALUES
    ('General'),
    ('Tech'),
    ('Funny'),
    ('News');
