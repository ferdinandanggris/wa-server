-- Create conversations table
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    assigned_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    status VARCHAR(20) DEFAULT 'open',
    last_customer_message_at TIMESTAMP WITH TIME ZONE,
    last_agent_message_at TIMESTAMP WITH TIME ZONE,
    is_24h_window_active BOOLEAN DEFAULT true,
    unread_count INTEGER DEFAULT 0,
    last_message_preview TEXT,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    closed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_conversations_company_id ON conversations(company_id);
CREATE INDEX idx_conversations_contact_id ON conversations(contact_id);
CREATE INDEX idx_conversations_assigned_agent_id ON conversations(assigned_agent_id);
CREATE INDEX idx_conversations_status ON conversations(status);
CREATE INDEX idx_conversations_24h_window ON conversations(is_24h_window_active, last_customer_message_at);

COMMENT ON TABLE conversations IS 'Chat conversations with 24-hour window tracking';
COMMENT ON COLUMN conversations.last_customer_message_at IS 'Timestamp of last customer message for 24h window';
COMMENT ON COLUMN conversations.is_24h_window_active IS 'True if within 24-hour customer message window';
COMMENT ON COLUMN conversations.status IS 'open, assigned, closed, escalated';