
-- name: CreateFeedFollow :many
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
VALUES(
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *
)
SELECT inserted_feed_follow.*, users.name, feeds.name
FROM inserted_feed_follow
JOIN users ON inserted_feed_follow.user_id = users.id
JOIN feeds ON inserted_feed_follow.feed_id = feeds.id;

-- name: GetFeedFollowForUser :many
SELECT *
FROM feed_follows
WHERE $1 = user_id;

-- name: DeleteFollowRecord :exec
DELETE FROM feed_follows
WHERE user_id = $1 AND feed_id = $2;
