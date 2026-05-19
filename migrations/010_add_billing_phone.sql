-- Add phone_number and conversation_category to billing_logs for Meta reconciliation.
ALTER TABLE billing_logs ADD COLUMN IF NOT EXISTS phone_number VARCHAR(50);
ALTER TABLE billing_logs ADD COLUMN IF NOT EXISTS conversation_category VARCHAR(20);
