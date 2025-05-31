-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, url, description, published_at, feed_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
) RETURNING *;

-- name: GetPostsForUser :many
SELECT * 
FROM posts
JOIN feed_follows
ON posts.feed_id = feed_follows.feed_id
AND $1 = feed_follows.user_id
LIMIT $2;

