-- Create templates table (Shared Library)
CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wa_template_id VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    language VARCHAR(10) NOT NULL,
    category VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    header_type VARCHAR(20),
    header_content TEXT,
    body_components JSONB,
    footer_text TEXT,
    buttons JSONB,
    is_verified BOOLEAN DEFAULT false,
    meta_status VARCHAR(50),
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_templates_wa_template_id ON templates(wa_template_id);
CREATE INDEX idx_templates_name ON templates(name);
CREATE INDEX idx_templates_category ON templates(category);
CREATE INDEX idx_templates_is_verified ON templates(is_verified);

COMMENT ON TABLE templates IS 'Shared template library accessible by all companies';
COMMENT ON COLUMN templates.is_verified IS 'Template verified by Meta';
COMMENT ON COLUMN templates.meta_status IS 'PENDING, APPROVED, REJECTED from Meta';