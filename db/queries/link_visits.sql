-- name: ListLinkVisitsRange :many
SELECT id, link_id, ip, user_agent, referer, status, created_at
FROM link_visits
ORDER BY id
LIMIT $2 OFFSET $1;

-- name: CountLinkVisits :one
SELECT count(*) FROM link_visits;

-- name: CreateLinkVisit :one
INSERT INTO link_visits (link_id, ip, user_agent, referer, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, link_id, ip, user_agent, referer, status, created_at;
