CREATE TABLE IF NOT EXISTS comment_reactions (
    id INTEGER PRIMARY KEY, -- comment_reactions_id
    user_id INTEGER NOT NULL,
    comment_id INTEGER NOT NULL,
    reaction_type TEXT NOT NULL CHECK (reaction_type IN ('like', 'dislike')),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (comment_id) REFERENCES comments(id),
    UNIQUE (user_id, comment_id)
);