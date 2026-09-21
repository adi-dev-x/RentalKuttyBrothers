--
-- Kutty Brothers — Rental Software
-- Full PostgreSQL schema (regenerated from the application code)
--
-- This file creates every table IF NOT EXISTS. For items, delivery_chelan,
-- delivery_items and transaction — the tables whose shape changed — it
-- creates the *original* column set, on purpose: on a database that already
-- has one of these tables in that original shape, CREATE TABLE IF NOT
-- EXISTS is a no-op and does not add/drop columns, so this file must match
-- what's already there. The migration_*.sql files are what actually alter
-- them:
--   migration_product_code.sql:
--     * items.product_code   (replaces main_code + sub_code) and items.hsn_code
--     * delivery_chelan: invoice_number, discount, guarantee_images, expiry_at,
--       vehicle_number, pass_entry_date/time, pass_exit_date/time
--     * delivery_items.damage
--     * repair_history                new table (damage + repair records)
--     * quotation / quotation_items   new tables (folded in from quotation_migration.sql)
--   migration_transaction_invoice.sql:
--     * transaction: drops the main_transaction relation (column + FK +
--       index) — it now links to its order directly via order_id — and
--       adds invoice_id, from_date, to_date, invoice_type
--
-- Run every file together, in this order, every time (all are idempotent):
--   psql -U postgres -d kuttybrothers -f schema.sql
--   psql -U postgres -d kuttybrothers -f migration_product_code.sql
--   psql -U postgres -d kuttybrothers -f migration_transaction_invoice.sql
-- (the app does this itself on every `go run` — see pkg/database/migrate.go)
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

CREATE SCHEMA IF NOT EXISTS public;
COMMENT ON SCHEMA public IS 'standard public schema';

-- uuid_generate_v4() is used as the default for every uuid primary key.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- gen_random_uuid() is used by the quotation tables.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

SET default_tablespace = '';
SET default_table_access_method = heap;


--
-- Generic API plumbing -------------------------------------------------------
--

CREATE TABLE IF NOT EXISTS public.api_registry (
    api_type   text NOT NULL,
    query_text text NOT NULL,
    CONSTRAINT api_registry_pkey PRIMARY KEY (api_type)
);

CREATE TABLE IF NOT EXISTS public.api_attributes (
    api_attributes_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    api               text NOT NULL,
    status            text,
    created_at        timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    keys              jsonb[],
    type              character varying,
    fields            jsonb,
    CONSTRAINT api_attributes_pkey PRIMARY KEY (api_attributes_id)
);


--
-- Attributes (brand / category / ItemMainType / ItemSubType lookups) ---------
--

CREATE TABLE IF NOT EXISTS public.attributes (
    attributes_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    type          text NOT NULL,
    name          text NOT NULL,
    status        text DEFAULT 'ACTIVE'::text,
    created_at    timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT attributes_pkey PRIMARY KEY (attributes_id),
    CONSTRAINT attributes_type_name_unique UNIQUE (type, name)
);


--
-- Users ----------------------------------------------------------------------
--

CREATE TABLE IF NOT EXISTS public.users (
    user_id     uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    username    text NOT NULL,
    password    text NOT NULL,
    designation text NOT NULL,
    email       text NOT NULL,
    created_at  timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_pkey PRIMARY KEY (user_id),
    CONSTRAINT users_username_key UNIQUE (username),
    CONSTRAINT users_email_key UNIQUE (email)
);


--
-- Customers ------------------------------------------------------------------
--

CREATE TABLE IF NOT EXISTS public.customer (
    customer_id   uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name          text NOT NULL,
    short_name    text NOT NULL,
    phone         text NOT NULL,
    type          text NOT NULL,
    gst           text NOT NULL,
    address       text NOT NULL,
    email         text,
    customer_flag text,
    status        text,
    created_at    timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT customer_pkey PRIMARY KEY (customer_id)
);

CREATE INDEX IF NOT EXISTS idx_customer_status ON public.customer (status);


--
-- Inventory (sites) ----------------------------------------------------------
--

CREATE TABLE IF NOT EXISTS public.inventory (
    inventory_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    sitename     text NOT NULL,
    status       text,
    created_at   timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT inventory_pkey PRIMARY KEY (inventory_id),
    CONSTRAINT inventory_sitename_key UNIQUE (sitename)
);


--
-- Items ----------------------------------------------------------------------
-- Created here in its original shape (item_code / sub_code / main_code, no
-- hsn_code) — see the file header. migration_product_code.sql turns
-- sub_code + main_code into a single product_code column and adds hsn_code.
--

CREATE TABLE IF NOT EXISTS public.items (
    item_id        uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    item_code      text NOT NULL,
    sub_code       text NOT NULL,
    item_name      text NOT NULL,
    item_main_type text NOT NULL,
    item_sub_type  text,
    brand          text,
    category       text,
    description    text,
    inventory_id   uuid,
    created_at     timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    main_code      character varying,
    CONSTRAINT items_pkey PRIMARY KEY (item_id),
    CONSTRAINT items_inventory_id_fkey FOREIGN KEY (inventory_id)
        REFERENCES public.inventory (inventory_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_items_category  ON public.items (category);
CREATE INDEX IF NOT EXISTS idx_items_item_code ON public.items (item_code);


--
-- Delivery chelan (the order header) ----------------------------------------
-- Created here in its original shape — see the file header.
-- migration_product_code.sql adds discount, invoice_number, guarantee_images,
-- vehicle_number, pass_entry/exit date+time and expiry_at.
--

CREATE TABLE IF NOT EXISTS public.delivery_chelan (
    delivery_id            uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    customer_id            uuid NOT NULL,
    inventory_id           uuid NOT NULL,
    advance_amount         integer NOT NULL,
    generated_amount       integer NOT NULL,
    placed_at              timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    declined_at            timestamp without time zone,
    status                 text,
    current_amount         integer,
    contact_name           text,
    contact_number         text,
    shipping_address       text,
    transaction_id         text,
    order_number           character varying,
    delivery_chelan_number character varying,
    CONSTRAINT delivery_chelan_pkey PRIMARY KEY (delivery_id),
    CONSTRAINT delivery_chelan_customer_id_fkey FOREIGN KEY (customer_id)
        REFERENCES public.customer (customer_id) ON DELETE CASCADE,
    CONSTRAINT delivery_chelan_inventory_id_fkey FOREIGN KEY (inventory_id)
        REFERENCES public.inventory (inventory_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_delivery_chelan_customer_id ON public.delivery_chelan (customer_id);
CREATE INDEX IF NOT EXISTS idx_delivery_chelan_status      ON public.delivery_chelan (status);
CREATE INDEX IF NOT EXISTS idx_delivery_chelan_placed_at   ON public.delivery_chelan (placed_at);


--
-- Delivery items (one row per rented unit on an order) -----------------------
-- Created here in its original shape — see the file header.
-- migration_product_code.sql adds the damage flag.
--

CREATE TABLE IF NOT EXISTS public.delivery_items (
    delivery_item_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    customer_id      uuid NOT NULL,
    inventory_id     uuid NOT NULL,
    rent_amount      integer NOT NULL,
    generated_amount integer NOT NULL,
    before_images    text[],
    after_images     text[],
    condition_out    text,
    condition_in     text,
    placed_at        timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    returned_at      timestamp without time zone,
    declined_at      timestamp without time zone,
    status           text,
    current_amount   integer,
    order_id         uuid,
    item_id          uuid,
    CONSTRAINT delivery_items_pkey PRIMARY KEY (delivery_item_id),
    CONSTRAINT delivery_items_customer_id_fkey FOREIGN KEY (customer_id)
        REFERENCES public.customer (customer_id) ON DELETE CASCADE,
    CONSTRAINT delivery_items_inventory_id_fkey FOREIGN KEY (inventory_id)
        REFERENCES public.inventory (inventory_id) ON DELETE CASCADE,
    CONSTRAINT delivery_items_order_id_fkey FOREIGN KEY (order_id)
        REFERENCES public.delivery_chelan (delivery_id) ON DELETE CASCADE,
    CONSTRAINT delivery_item_id_fkey FOREIGN KEY (item_id)
        REFERENCES public.items (item_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_delivery_items_order_id    ON public.delivery_items (order_id);
CREATE INDEX IF NOT EXISTS idx_delivery_items_item_id     ON public.delivery_items (item_id);
CREATE INDEX IF NOT EXISTS idx_delivery_items_customer_id ON public.delivery_items (customer_id);
CREATE INDEX IF NOT EXISTS idx_delivery_items_status      ON public.delivery_items (status);


--
-- Item status history --------------------------------------------------------
--

CREATE TABLE IF NOT EXISTS public.item_status_history (
    history_id       uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    item_id          uuid NOT NULL,
    old_status       text,
    new_status       text,
    changed_at       timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    changed_by       uuid,
    delivery_id      uuid,
    delivery_item_id uuid,
    notes            text,
    CONSTRAINT item_status_history_pkey PRIMARY KEY (history_id),
    CONSTRAINT item_status_history_item_id_fkey FOREIGN KEY (item_id)
        REFERENCES public.items (item_id) ON DELETE CASCADE,
    CONSTRAINT item_status_history_changed_by_fkey FOREIGN KEY (changed_by)
        REFERENCES public.users (user_id) ON DELETE SET NULL,
    CONSTRAINT item_status_history_delivery_id_fkey FOREIGN KEY (delivery_id)
        REFERENCES public.delivery_chelan (delivery_id) ON DELETE SET NULL,
    CONSTRAINT item_status_history_delivery_item_id_fkey FOREIGN KEY (delivery_item_id)
        REFERENCES public.delivery_items (delivery_item_id) ON DELETE SET NULL
);


--
-- Repair / damage history ----------------------------------------------------
-- item_id and order_item_id are stored as text: the application joins them with
-- items.item_id::TEXT and delivery_items.delivery_item_id::TEXT.
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
-- Transactions ---------------------------------------------------------------
--

CREATE TABLE IF NOT EXISTS public.main_transaction (
    id       integer GENERATED BY DEFAULT AS IDENTITY NOT NULL,
    order_id uuid NOT NULL,
    status   character varying(10) NOT NULL,
    amount   numeric(12,2),
    CONSTRAINT main_transaction_pkey PRIMARY KEY (id),
    CONSTRAINT main_transaction_status_check
        CHECK (((status)::text = ANY ((ARRAY['PENDING'::character varying, 'COMPLETED'::character varying])::text[]))),
    CONSTRAINT main_transaction_order_id_fkey FOREIGN KEY (order_id)
        REFERENCES public.delivery_chelan (delivery_id)
);

-- Created here in its original shape (still related to main_transaction) —
-- see the file header. migration_transaction_invoice.sql drops that
-- relation and adds invoice_id, from_date, to_date, invoice_type.
CREATE TABLE IF NOT EXISTS public.transaction (
    id                  integer GENERATED BY DEFAULT AS IDENTITY NOT NULL,
    main_transaction_id integer NOT NULL,
    order_id            uuid,
    status              character varying(10) NOT NULL,
    amount              numeric(12,2) NOT NULL,
    transaction_type    character varying(10) NOT NULL,
    image               text,
    CONSTRAINT transaction_pkey PRIMARY KEY (id),
    CONSTRAINT transaction_status_check
        CHECK (((status)::text = ANY ((ARRAY['PENDING'::character varying, 'COMPLETED'::character varying, 'FAILED'::character varying])::text[]))),
    CONSTRAINT transaction_main_transaction_id_fkey FOREIGN KEY (main_transaction_id)
        REFERENCES public.main_transaction (id) ON DELETE CASCADE,
    CONSTRAINT transaction_order_id_fkey FOREIGN KEY (order_id)
        REFERENCES public.delivery_chelan (delivery_id)
);

CREATE INDEX IF NOT EXISTS idx_main_transaction_order_id ON public.main_transaction (order_id);
-- No index on transaction.main_transaction_id here: that column and its
-- relation are dropped by migration_transaction_invoice.sql, which creates
-- idx_transaction_order_id / idx_transaction_invoice_id instead.


--
-- Quotations -----------------------------------------------------------------
--

CREATE TABLE IF NOT EXISTS public.quotation (
    quotation_id     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id      uuid REFERENCES public.customer (customer_id) ON DELETE SET NULL,
    contact_name     varchar(255) NOT NULL DEFAULT '',
    contact_number   varchar(50)  NOT NULL DEFAULT '',
    shipping_address text         NOT NULL DEFAULT '',
    product_code     text         NOT NULL DEFAULT '',
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

CREATE INDEX IF NOT EXISTS idx_quotation_customer_id       ON public.quotation (customer_id);
CREATE INDEX IF NOT EXISTS idx_quotation_items_quotation_id ON public.quotation_items (quotation_id);


--
-- Generic API seed data ------------------------------------------------------
-- These rows drive /irrl/genericApiUnjoin/:ApiType and /irrl/genericApiJoin/:ApiType.
-- productMain and productSingle are the ones that changed for product_code.
--

INSERT INTO public.api_registry (api_type, query_text) VALUES
('customer',
 'select  customer_id,name  from  customer'),

('customerlist',
 'select  *  from  customer'),

('listProductCount',
 'SELECT COUNT(category) AS count FROM public.items'),

('listPaymentCountPending',
 'SELECT COUNT(status) AS count FROM public.main_transaction where status=''PENDING'''),

('listTransactionCount',
 'SELECT COUNT(status) AS count FROM public.main_transaction'),

('mainTransaction',
 'SELECT id AS transaction_id,amount,status  FROM public.main_transaction'),

('subTransaction',
 'SELECT * FROM public.transaction'),

('topRentedProduct',
 'SELECT item_name, COUNT(*) AS rented_count FROM public.items WHERE category = ''RENTED'' GROUP BY item_name ORDER BY rented_count DESC'),

('itemRetrive',
 'SELECT item_name, item_id, item_code FROM public.items WHERE category = ''AVAILABLE'''),

('productSingle',
 'SELECT * FROM public.items '),

('productMain',
 'SELECT product_code, MAX(item_name) AS item_name, MAX(item_sub_type) AS item_sub_type, MAX(item_main_type) AS item_main_type, MAX(description) AS description, MAX(brand) AS brand, MAX(hsn_code) AS hsn_code, COUNT(*) FILTER (WHERE category = ''REPAIRING'') AS not_initiated_count, COUNT(*) FILTER (WHERE category = ''RENTED'') AS rented_count, COUNT(*) FILTER (WHERE category = ''DAMAGED'') AS damaged_count, COUNT(*) FILTER (WHERE category = ''AVAILABLE'') AS available_count, COUNT(*) FILTER (WHERE category = ''EXPIRED'') AS worn_out_count, COUNT(*) FILTER (WHERE category = ''RESERVED'') AS reserved_count, COUNT(*) FILTER (WHERE category = ''BLOCKED'') AS blocked_count, COUNT(*) FILTER (WHERE category = ''PENDING'') AS pending_count FROM public.items GROUP BY product_code ORDER BY product_code'),

('listOrders',
 'SELECT dc.*, customer.name AS customer_name FROM delivery_chelan dc INNER JOIN customer ON dc.customer_id = customer.customer_id where dc.status!=''DELETED'''),

('listAllOrders',
 'SELECT di.*, c.customer_id, c.name AS customer_name, dc.contact_name, dc.contact_number, dc.shipping_address FROM public.delivery_items di INNER JOIN public.customer c ON di.customer_id = c.customer_id INNER JOIN public.delivery_chelan dc ON di.order_id = dc.delivery_id'),

('orderDetails',
 'SELECT i.item_name, i.item_code, i.product_code, c.name AS customer_name, c.gst AS customer_gst, dc.delivery_chelan_number, dc.order_number, di.* FROM public.delivery_items di INNER JOIN public.items i ON di.item_id = i.item_id INNER JOIN public.customer c ON di.customer_id = c.customer_id INNER JOIN public.delivery_chelan dc ON di.order_id = dc.delivery_id')
ON CONFLICT (api_type) DO NOTHING;

INSERT INTO public.api_attributes (api, type, fields)
SELECT v.api, v.type, v.fields::jsonb
FROM (VALUES
    ('productSingle',        'api',    '{"key": "category", "table": "items", "condition": "="}'),
    ('productSingle',        'api',    '{"key": "product_code", "table": "items", "condition": "="}'),
    ('listProductCount',     'api',    '{"key": "category", "table": "items", "condition": "="}'),
    ('listTransactionCount', 'api',    '{"key": "status", "table": "main_transaction", "condition": "="}'),
    ('mainTransaction',      'api',    '{"key": "order_id", "table": "main_transaction", "condition": "="}'),
    ('subTransaction',       'api',    '{"key": "order_id", "table": "transaction", "condition": "="}'),
    ('subTransaction',       'api',    '{"key": "invoice_id", "table": "transaction", "condition": "="}'),
    ('orderDetails',         'api',    '{"key": "order_id", "table": "delivery_items", "condition": "="}'),
    ('customerlist',         'api',    '{"key": "customer_flag", "condition": "="}'),
    ('customerlist',         'api',    '{"key": "status", "condition": "="}'),
    ('sd-rt-P1-u',           'update', '{"id": "item_id", "key": "category", "table": "items"}'),
    ('customerSingle',       'insert', '{"gst": "string", "name": "string", "type": "string", "email": "string", "phone": "string", "status": "string", "address": "string", "short_name": "string", "customer_flag": "string"}')
) AS v(api, type, fields)
WHERE NOT EXISTS (
    SELECT 1 FROM public.api_attributes a
    WHERE a.api = v.api AND a.type = v.type AND a.fields = v.fields::jsonb
);


--
-- Privileges -----------------------------------------------------------------
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO PUBLIC;

--
-- End of schema
--
