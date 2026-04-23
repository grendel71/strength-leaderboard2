-- name: GetAthleteByID :one
SELECT * FROM athletes WHERE id = $1;

-- name: ListAthletes :many
SELECT * FROM athletes ORDER BY total DESC NULLS LAST;

-- name: ListAthletesByGender :many
SELECT * FROM athletes WHERE gender = $1 ORDER BY total DESC NULLS LAST;

-- name: ListAthletesSorted :many
SELECT * FROM athletes
ORDER BY
    CASE WHEN @sort_field::text = 'squat' THEN squat END DESC NULLS LAST,
    CASE WHEN @sort_field::text = 'bench' THEN bench END DESC NULLS LAST,
    CASE WHEN @sort_field::text = 'deadlift' THEN deadlift END DESC NULLS LAST,
    CASE WHEN @sort_field::text = 'ohp' THEN ohp END DESC NULLS LAST,
    CASE WHEN @sort_field::text = 'total' OR @sort_field::text = '' THEN total END DESC NULLS LAST;

-- name: ListAthletesSortedByGender :many
SELECT * FROM athletes
WHERE gender = $1
ORDER BY
    CASE WHEN @sort_field::text = 'squat' THEN squat END DESC NULLS LAST,
    CASE WHEN @sort_field::text = 'bench' THEN bench END DESC NULLS LAST,
    CASE WHEN @sort_field::text = 'deadlift' THEN deadlift END DESC NULLS LAST,
    CASE WHEN @sort_field::text = 'ohp' THEN ohp END DESC NULLS LAST,
    CASE WHEN @sort_field::text = 'total' OR @sort_field::text = '' THEN total END DESC NULLS LAST;

-- name: CreateAthlete :one
INSERT INTO athletes (name, gender, body_weight, squat, bench, deadlift, total, ohp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateAthlete :one
UPDATE athletes SET
    name = $2,
    gender = $3,
    body_weight = $4,
    avatar_url = $5,
    bio = $6,
    squat = $7,
    bench = $8,
    deadlift = $9,
    total = $10,
    ohp = $11,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (username, password, athlete_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: GetSession :one
SELECT s.*, u.username, u.athlete_id
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.id = $1 AND s.expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < NOW();
