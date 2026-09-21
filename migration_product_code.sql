--
-- Kutty Brothers — Rental Software
-- Upgrade migration: brings an existing database up to schema.sql
--
--   1. items: main_code + sub_code  ->  product_code, plus hsn_code
--   2. delivery_chelan: invoice/discount/guarantee/gate-pass columns
--   3. delivery_items: damage flag
--   4. repair_history table
--   5. quotation + quotation_items tables
--   6. api_registry / api_attributes rows rewritten for product_code
--
-- Run once:  psql -U postgres -d kuttybrothers -f migration_product_code.sql
-- It is wrapped in a transaction, so it either applies fully or not at all.
--

BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;

--
-- 1. items ------------------------------------------------------------------
--

ALTER TABLE public.items ADD COLUMN IF NOT EXISTS product_code text;
ALTER TABLE public.items ADD COLUMN IF NOT EXISTS hsn_code     text;

-- Carry the old codes over: sub_code was the code every query grouped/filtered
-- on, so it wins; main_code is the fallback when sub_code is empty. Guarded by
-- column-existence checks (rather than referencing sub_code/main_code
-- directly) because this file also runs, no-op, against a database that was
-- built straight from schema.sql and never had those columns — PL/pgSQL only
-- plans an embedded statement the first time its branch actually executes, so
-- the UPDATE below is never parsed against a schema where it wouldn't apply.
DO $$
DECLARE
    has_sub_code  boolean;
    has_main_code boolean;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'items' AND column_name = 'sub_code'
    ) INTO has_sub_code;
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'items' AND column_name = 'main_code'
    ) INTO has_main_code;

    IF has_sub_code AND has_main_code THEN
        UPDATE public.items
        SET product_code = COALESCE(NULLIF(TRIM(sub_code), ''), NULLIF(TRIM(main_code), ''), item_code)
        WHERE product_code IS NULL;
    ELSIF has_sub_code THEN
        UPDATE public.items
        SET product_code = COALESCE(NULLIF(TRIM(sub_code), ''), item_code)
        WHERE product_code IS NULL;
    ELSIF has_main_code THEN
        UPDATE public.items
        SET product_code = COALESCE(NULLIF(TRIM(main_code), ''), item_code)
        WHERE product_code IS NULL;
    END IF;
END $$;

UPDATE public.items SET product_code = item_code WHERE product_code IS NULL;

ALTER TABLE public.items ALTER COLUMN product_code SET NOT NULL;

ALTER TABLE public.items DROP COLUMN IF EXISTS sub_code;
ALTER TABLE public.items DROP COLUMN IF EXISTS main_code;

CREATE INDEX IF NOT EXISTS idx_items_product_code ON public.items (product_code);
CREATE INDEX IF NOT EXISTS idx_items_category     ON public.items (category);
CREATE INDEX IF NOT EXISTS idx_items_item_code    ON public.items (item_code);

--
-- 2. delivery_chelan --------------------------------------------------------
--

ALTER TABLE public.delivery_chelan ADD COLUMN IF NOT EXISTS discount         integer DEFAULT 0;
ALTER TABLE public.delivery_chelan ADD COLUMN IF NOT EXISTS invoice_number   character varying;
ALTER TABLE public.delivery_chelan ADD COLUMN IF NOT EXISTS guarantee_images text[];
ALTER TABLE public.delivery_chelan ADD COLUMN IF NOT EXISTS vehicle_number   text;
ALTER TABLE public.delivery_chelan ADD COLUMN IF NOT EXISTS pass_entry_date  text;
ALTER TABLE public.delivery_chelan ADD COLUMN IF NOT EXISTS pass_entry_time  text;
ALTER TABLE public.delivery_chelan ADD COLUMN IF NOT EXISTS pass_exit_date   text;
ALTER TABLE public.delivery_chelan ADD COLUMN IF NOT EXISTS pass_exit_time   text;
ALTER TABLE public.delivery_chelan ADD COLUMN IF NOT EXISTS expiry_at        timestamp without time zone;

UPDATE public.delivery_chelan SET discount = 0 WHERE discount IS NULL;

CREATE INDEX IF NOT EXISTS idx_delivery_chelan_customer_id ON public.delivery_chelan (customer_id);
CREATE INDEX IF NOT EXISTS idx_delivery_chelan_status      ON public.delivery_chelan (status);
CREATE INDEX IF NOT EXISTS idx_delivery_chelan_placed_at   ON public.delivery_chelan (placed_at);

--
-- 3. delivery_items ---------------------------------------------------------
--

ALTER TABLE public.delivery_items ADD COLUMN IF NOT EXISTS damage boolean DEFAULT false;
UPDATE public.delivery_items SET damage = false WHERE damage IS NULL;

CREATE INDEX IF NOT EXISTS idx_delivery_items_order_id    ON public.delivery_items (order_id);
CREATE INDEX IF NOT EXISTS idx_delivery_items_item_id     ON public.delivery_items (item_id);
CREATE INDEX IF NOT EXISTS idx_delivery_items_customer_id ON public.delivery_items (customer_id);
CREATE INDEX IF NOT EXISTS idx_delivery_items_status      ON public.delivery_items (status);

--
-- 4. repair_history ---------------------------------------------------------
--

CREATE TABLE IF NOT EXISTS public.repair_history (
    id            bigserial NOT NULL,
    item_id       text NOT NULL,
    order_item_id text,
    damage_images text[],
    amount        integer DEFAULT 0,
    description   text,
    created_at    timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT repair_history_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_repair_history_item_id       ON public.repair_history (item_id);
CREATE INDEX IF NOT EXISTS idx_repair_history_order_item_id ON public.repair_history (order_item_id);
CREATE INDEX IF NOT EXISTS idx_repair_history_created_at    ON public.repair_history (created_at);

--
-- 5. quotation --------------------------------------------------------------
--

CREATE TABLE IF NOT EXISTS public.quotation (
    quotation_id     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id      uuid REFERENCES public.customer (customer_id) ON DELETE SET NULL,
    contact_name     varchar(255) NOT NULL DEFAULT '',
    contact_number   varchar(50)  NOT NULL DEFAULT '',
    shipping_address text         NOT NULL DEFAULT '',
    product_code     text           NOT NULL DEFAULT '',
    quotation_number varchar(10)  NOT NULL UNIQUE,
    total_amount     integer      NOT NULL DEFAULT 0,
    placed_at        timestamptz  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.quotation_items (
    quotation_item_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    quotation_id      uuid         NOT NULL REFERENCES public.quotation (quotation_id) ON DELETE CASCADE,
    item_name         varchar(255) NOT NULL,
    description       text,
    rent_amount       integer      NOT NULL DEFAULT 0,
    start_date        varchar(20),
    end_date          varchar(20),
    created_at        timestamptz  NOT NULL DEFAULT NOW()
);

-- In case an older copy of quotation_migration.sql (without product_code)
-- already created this table elsewhere before this file existed.
ALTER TABLE public.quotation ADD COLUMN IF NOT EXISTS product_code text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_quotation_customer_id        ON public.quotation (customer_id);
CREATE INDEX IF NOT EXISTS idx_quotation_items_quotation_id ON public.quotation_items (quotation_id);

--
-- 6. Generic API rows -------------------------------------------------------
--

UPDATE public.api_registry
SET query_text = 'SELECT product_code, MAX(item_name) AS item_name, MAX(item_sub_type) AS item_sub_type, MAX(item_main_type) AS item_main_type, MAX(description) AS description, MAX(brand) AS brand, MAX(hsn_code) AS hsn_code, COUNT(*) FILTER (WHERE category = ''REPAIRING'') AS not_initiated_count, COUNT(*) FILTER (WHERE category = ''RENTED'') AS rented_count, COUNT(*) FILTER (WHERE category = ''DAMAGED'') AS damaged_count, COUNT(*) FILTER (WHERE category = ''AVAILABLE'') AS available_count, COUNT(*) FILTER (WHERE category = ''EXPIRED'') AS worn_out_count, COUNT(*) FILTER (WHERE category = ''RESERVED'') AS reserved_count, COUNT(*) FILTER (WHERE category = ''BLOCKED'') AS blocked_count, COUNT(*) FILTER (WHERE category = ''PENDING'') AS pending_count FROM public.items GROUP BY product_code ORDER BY product_code'
WHERE api_type = 'productMain';

UPDATE public.api_registry
SET query_text = 'SELECT i.item_name, i.item_code, i.product_code, c.name AS customer_name, c.gst AS customer_gst, dc.delivery_chelan_number, dc.order_number, di.* FROM public.delivery_items di INNER JOIN public.items i ON di.item_id = i.item_id INNER JOIN public.customer c ON di.customer_id = c.customer_id INNER JOIN public.delivery_chelan dc ON di.order_id = dc.delivery_id'
WHERE api_type = 'orderDetails';

-- The filter key the frontend passes to productSingle is now product_code.
UPDATE public.api_attributes
SET fields = '{"key": "product_code", "table": "items", "condition": "="}'::jsonb
WHERE fields->>'key' = 'sub_code';

COMMIT;
