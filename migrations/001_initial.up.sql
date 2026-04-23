CREATE TABLE athletes (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) UNIQUE NOT NULL,
    gender      VARCHAR(10) DEFAULT 'male',
    body_weight DECIMAL(6,2),
    avatar_url  TEXT,
    squat       DECIMAL(7,2),
    bench       DECIMAL(7,2),
    deadlift    DECIMAL(7,2),
    total       DECIMAL(8,2),
    ohp         DECIMAL(7,2),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE users (
    id          SERIAL PRIMARY KEY,
    username    VARCHAR(50) UNIQUE NOT NULL,
    password    TEXT NOT NULL,
    athlete_id  INT REFERENCES athletes(id),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE sessions (
    id         VARCHAR(64) PRIMARY KEY,
    user_id    INT REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_athletes_total ON athletes(total DESC NULLS LAST);
CREATE INDEX idx_athletes_gender ON athletes(gender);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
CREATE INDEX idx_users_athlete ON users(athlete_id);
