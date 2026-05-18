-- Create companies table (Multi-tenant isolation)
CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    phone_number VARCHAR(50),
    address TEXT,
    is_active BOOLEAN DEFAULT true,
    quota_limit INTEGER DEFAULT 50000,
    quota_used INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_companies_code ON companies(code);
CREATE INDEX idx_companies_is_active ON companies(is_active);

COMMENT ON TABLE companies IS 'Multi-tenant companies table - each company is a tenant';
COMMENT ON COLUMN companies.code IS 'Unique company code for identification';
COMMENT ON COLUMN companies.quota_limit IS 'WhatsApp message quota limit per company';
COMMENT ON COLUMN companies.quota_used IS 'Current quota usage for billing';