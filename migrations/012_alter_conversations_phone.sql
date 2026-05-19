ALTER TABLE conversations ALTER COLUMN company_id DROP NOT NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS phone_number VARCHAR(50);
CREATE INDEX IF NOT EXISTS idx_conversations_phone ON conversations(phone_number);
