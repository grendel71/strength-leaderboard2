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

-- name: ListBonusLiftDefinitions :many
SELECT * FROM bonus_lift_definitions ORDER BY name ASC;

-- name: GetBonusLiftDefinitionByName :one
SELECT * FROM bonus_lift_definitions WHERE name = $1;

-- name: CreateBonusLiftDefinition :one
INSERT INTO bonus_lift_definitions (name)
VALUES ($1)
RETURNING *;

-- name: GetAthleteBonusLifts :many
SELECT bld.name, abl.value, bld.id as definition_id
FROM athlete_bonus_lifts abl
JOIN bonus_lift_definitions bld ON abl.lift_definition_id = bld.id
WHERE abl.athlete_id = $1;

-- name: UpsertAthleteBonusLift :exec
INSERT INTO athlete_bonus_lifts (athlete_id, lift_definition_id, value)
VALUES ($1, $2, $3)
ON CONFLICT (athlete_id, lift_definition_id) DO UPDATE SET value = $3;

-- name: DeleteAthleteBonusLift :exec
DELETE FROM athlete_bonus_lifts
WHERE athlete_id = $1 AND lift_definition_id = $2;

-- name: ListAthletesByBonusLift :many
SELECT a.*, abl.value as lift_value, bld.name as lift_name
FROM athletes a
JOIN athlete_bonus_lifts abl ON a.id = abl.athlete_id
JOIN bonus_lift_definitions bld ON abl.lift_definition_id = bld.id
WHERE bld.id = $1
AND (a.gender = $2 OR $2 = '')
ORDER BY abl.value DESC;

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
