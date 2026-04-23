-- Add bio column to athletes
ALTER TABLE athletes ADD COLUMN IF NOT EXISTS bio TEXT;