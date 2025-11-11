-- +goose Up
ALTER TABLE feed_follows RENAME COLUMN feeds_id TO feed_id;

-- +goose Down
ALTER TABLE feed_follows RENAME COLUMN feed_id TO feeds_id;