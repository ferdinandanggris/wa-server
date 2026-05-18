-- Create messages table
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    message_id VARCHAR(100) UNIQUE NOT NULL,
    direction VARCHAR(10) NOT NULL,
    message_type VARCHAR(20) NOT NULL,
    content TEXT,
    template_id UUID REFERENCES templates(id),
    template_params JSONB,
    media_url TEXT,
    status VARCHAR(20) DEFAULT 'sent',
    wa_status VARCHAR(50),
    sent_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    read_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_message_id ON messages(message_id);
CREATE INDEX idx_messages_direction ON messages(direction);
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_messages_template_id ON messages(template_id);

COMMENT ON TABLE messages IS 'Message storage with tenant isolation via conversation';
COMMENT ON COLUMN messages.direction IS 'inbound, outbound';
COMMENT ON COLUMN messages.message_type IS 'text, image, video, document, template';
COMMENT ON COLUMN messages.status IS 'pending, sent, delivered, read, failed';
COMMENT ON COLUMN messages.wa_status IS 'WhatsApp API status state';