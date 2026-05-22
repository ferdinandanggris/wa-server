-- Add UNIQUE constraint on contact_id + phone_number to prevent duplicate conversations

-- First, deduplicate: keep only the most recent conversation per (contact_id, phone_number)
DELETE FROM conversations a USING (
  SELECT id, ROW_NUMBER() OVER (
    PARTITION BY contact_id, phone_number ORDER BY updated_at DESC, created_at DESC
  ) as rn
  FROM conversations
) b
WHERE a.id = b.id AND b.rn > 1;

-- Add unique constraint
ALTER TABLE conversations ADD CONSTRAINT uq_conversation_contact_phone UNIQUE(contact_id, phone_number);
