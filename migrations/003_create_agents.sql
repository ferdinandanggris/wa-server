-- Create agents table (CS Agents)
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'offline',
    max_concurrent_chats INTEGER DEFAULT 10,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_agents_company_id ON agents(company_id);
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_email ON agents(email);

COMMENT ON TABLE agents IS 'Customer service agents with presence tracking';
COMMENT ON COLUMN agents.status IS 'online, offline, busy, away';
COMMENT ON COLUMN agents.max_concurrent_chats IS 'Maximum concurrent chat sessions';