CREATE TABLE post_categories (
    id INTEGER PRIMARY KEY, -- post_categories_id
    post_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    FOREIGN KEY (post_id) REFERENCES posts(id),
    FOREIGN KEY (category_id) REFERENCES categories(id),
    UNIQUE (post_id, category_id) -- Prevent duplicate post-category pairs inside TABLE "post_categories"
);