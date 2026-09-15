CREATE TABLE post_reactions (
    id INTEGER PRIMARY KEY, -- post_reactions_id
    user_id INTEGER NOT NULL,
    post_id INTEGER NOT NULL,
    reaction_type TEXT NOT NULL CHECK (reaction_type IN ('like', 'dislike')),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (post_id) REFERENCES posts(id),
    UNIQUE (user_id, post_id)
);