-- name: ListLinksRange :many
SELECT id, original_url, short_name, short_url, created_at
FROM links
ORDER BY id
LIMIT $2 OFFSET $1;

-- name: CountLinks :one
SELECT count(*) FROM links;

-- name: GetLink :one
SELECT id, original_url, short_name, short_url, created_at
FROM links
WHERE id = $1;

-- name: CreateLink :one
INSERT INTO links (original_url, short_name, short_url)
VALUES ($1, $2, $3)
RETURNING id, original_url, short_name, short_url, created_at;

-- name: UpdateLink :one
UPDATE links
SET original_url = $2,
    short_name = $3,
    short_url = $4
WHERE id = $1
RETURNING id, original_url, short_name, short_url, created_at;

-- name: DeleteLink :one
DELETE FROM links
WHERE id = $1
RETURNING id;

