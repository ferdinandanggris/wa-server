ALTER TABLE messages ADD COLUMN IF NOT EXISTS message_timestamp BIGINT;
CREATE INDEX IF NOT EXISTS idx_messages_message_timestamp ON messages(message_timestamp);
