-- +goose Up
CREATE TABLE posts(
    id uuid PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    title TEXT UNIQUE NOT NULL,
    url TEXT UNIQUE NOT NULL,
    description TEXT UNIQUE NOT NULL,
    published_at TEXT NOT NULL,
    feed_id uuid REFERENCES feeds (id) ON DELETE CASCADE NOT NULL
);

-- +goose Down
DROP TABLE posts;