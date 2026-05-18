-- Create contacts table (WhatsApp customers)
CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    wa_id VARCHAR(100) NOT NULL,
    phone_number VARCHAR(50) NOT NULL,
    name VARCHAR(255),
    profile_picture_url TEXT,
    is_blocked BOOLEAN DEFAULT false,
    last_seen_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(company_id, wa_id)
);

CREATE INDEX idx_contacts_company_id ON contacts(company_id);
CREATE INDEX idx_contacts_wa_id ON contacts(wa_id);
CREATE INDEX idx_contacts_phone_number ON contacts(phone_number);

COMMENT ON TABLE contacts IS 'WhatsApp contacts with tenant isolation';
COMMENT ON COLUMN contacts.company_id IS 'Foreign key for tenant isolation';
COMMENT ON COLUMN contacts.wa_id IS 'WhatsApp Business API user ID';
COMMENT ON COLUMN contacts.is_blocked IS 'Contact blocked by company';