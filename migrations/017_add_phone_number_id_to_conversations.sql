ALTER TABLE conversations ADD COLUMN phone_number_id UUID REFERENCES phone_numbers(id);

CREATE INDEX idx_conversations_phone_number_id ON conversations(phone_number_id);
