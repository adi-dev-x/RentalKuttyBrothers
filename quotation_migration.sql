-- Quotation tables migration

CREATE TABLE IF NOT EXISTS quotation (
    quotation_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id      UUID REFERENCES customer(customer_id) ON DELETE SET NULL,
    contact_name     VARCHAR(255) NOT NULL DEFAULT '',
    contact_number   VARCHAR(50)  NOT NULL DEFAULT '',
    shipping_address TEXT         NOT NULL DEFAULT '',
    quotation_number VARCHAR(10)  NOT NULL UNIQUE,
    total_amount     INT          NOT NULL DEFAULT 0,
    placed_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quotation_items (
    quotation_item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quotation_id      UUID         NOT NULL REFERENCES quotation(quotation_id) ON DELETE CASCADE,
    item_name         VARCHAR(255) NOT NULL,
    description       TEXT,
    rent_amount       INT          NOT NULL DEFAULT 0,
    start_date        VARCHAR(20),
    end_date          VARCHAR(20),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quotation_items_quotation_id ON quotation_items(quotation_id);
CREATE INDEX IF NOT EXISTS idx_quotation_customer_id ON quotation(customer_id);
