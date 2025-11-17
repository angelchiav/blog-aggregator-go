-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetFeed :many
SELECT *
FROM feeds
ORDER BY created_at ASC;

-- name: CreateFeedFollow :one
WITH inserted AS (
  INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
  VALUES ($1, $2, $3, $4, $5)
  RETURNING id, created_at, updated_at, user_id, feed_id
)
SELECT
  i.id,
  i.created_at,
  i.updated_at,
  i.user_id,
  i.feed_id,
  u.name AS user_name,
  f.name AS feed_name
FROM inserted i
JOIN users u ON u.id = i.user_id
JOIN feeds f ON f.id = i.feed_id;

-- name: GetFeedByURL :one
SELECT id, created_at, updated_at, name, url, user_id
FROM feeds
WHERE url = $1;

-- name: GetFeedFollowsForUser :many
SELECT
    ff.id,
    ff.created_at,
    ff.updated_at,
    ff.user_id,
    ff.feed_id,
    u.name AS user_name,
    f.name AS feed_name
FROM feed_follows AS ff
JOIN users AS u ON u.id = ff.user_id
JOIN feeds AS f ON f.id = ff.feed_id
WHERE ff.user_id = $1
ORDER BY ff.created_at ASC;

-- name: DeleteFeedFollowRecord :exec
DELETE FROM feed_follows
WHERE user_id = $1
  AND feed_id = $2;

-- name: MarkFeedFetched :exec
UPDATE feeds
SET last_fetched_at = NOW(),
    updated_at      = NOW()
WHERE id = $1;

-- name: GetNextFeedToFetch :one
SELECT *
FROM feeds
ORDER BY last_fetched_at NULLS FIRST, created_at ASC
LIMIT 1;