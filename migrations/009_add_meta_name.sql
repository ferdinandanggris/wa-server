-- Add meta_name column to templates table for Meta template name storage.
-- meta_name stores the name used when creating the template in Meta's API.
ALTER TABLE templates ADD COLUMN IF NOT EXISTS meta_name VARCHAR(255);
