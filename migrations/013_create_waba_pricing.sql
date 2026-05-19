CREATE TABLE IF NOT EXISTS waba_pricing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    waba_id VARCHAR(100) NOT NULL,
    phone_number VARCHAR(50) NOT NULL,
    pricing_category VARCHAR(20) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    volume INT NOT NULL DEFAULT 0,
    cost NUMERIC(10,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(waba_id, phone_number, pricing_category, start_time)
);

CREATE INDEX IF NOT EXISTS idx_waba_pricing_phone ON waba_pricing(phone_number);
CREATE INDEX IF NOT EXISTS idx_waba_pricing_waba ON waba_pricing(waba_id);
CREATE INDEX IF NOT EXISTS idx_waba_pricing_time ON waba_pricing(start_time, end_time);

ALTER TABLE phone_numbers ADD COLUMN IF NOT EXISTS last_sync_pricing BIGINT;
