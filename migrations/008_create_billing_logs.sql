-- Create billing_logs table (Template usage tracking per company)
CREATE TABLE IF NOT EXISTS billing_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    template_cost DECIMAL(10, 4) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_billing_logs_company_id ON billing_logs(company_id);
CREATE INDEX idx_billing_logs_template_id ON billing_logs(template_id);
CREATE INDEX idx_billing_logs_created_at ON billing_logs(created_at);
CREATE INDEX idx_billing_logs_company_date ON billing_logs(company_id, DATE(created_at));

COMMENT ON TABLE billing_logs IS 'Template usage billing with tenant isolation';
COMMENT ON COLUMN billing_logs.template_cost IS 'Cost per template message';
COMMENT ON COLUMN billing_logs.company_id IS 'Foreign key for tenant isolation - for accurate per-company billing';