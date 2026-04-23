ALTER TABLE bonus_lift_definitions
ADD COLUMN IF NOT EXISTS enable_distance BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE bonus_lift_definitions
ADD COLUMN IF NOT EXISTS enable_reps BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE athlete_bonus_lifts
ALTER COLUMN value DROP NOT NULL;

ALTER TABLE athlete_bonus_lifts
ADD COLUMN IF NOT EXISTS distance DECIMAL(7,2);

ALTER TABLE athlete_bonus_lifts
ADD COLUMN IF NOT EXISTS reps INT;
