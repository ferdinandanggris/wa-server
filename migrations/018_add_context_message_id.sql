ALTER TABLE messages ADD COLUMN context_message_id UUID REFERENCES messages(id);
CREATE INDEX idx_messages_context_message_id ON messages(context_message_id);
