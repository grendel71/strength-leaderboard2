CREATE TABLE IF NOT EXISTS bonus_lift_definitions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS athlete_bonus_lifts (
    athlete_id INT REFERENCES athletes(id) ON DELETE CASCADE,
    lift_definition_id INT REFERENCES bonus_lift_definitions(id) ON DELETE CASCADE,
    value DECIMAL(7,2) NOT NULL,
    PRIMARY KEY (athlete_id, lift_definition_id)
);
