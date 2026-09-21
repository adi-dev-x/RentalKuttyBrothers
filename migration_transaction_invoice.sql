--
-- Kutty Brothers — Rental Software
-- Upgrade migration: transaction no longer relates to main_transaction
--
--   1. transaction: drop the main_transaction relation (FK + index + column).
--      It now links to its order directly via the order_id column that was
--      already on the table but previously left unpopulated by
--      AddSubTransaction.
--   2. transaction: add invoice_id, from_date, to_date, invoice_type.
--      invoice_id is always generated server-side (see
--      repository.generateTransactionInvoiceID) — clients never set it.
--      invoice_type was originally added as tax_type; if a database already
--      has that column (from an earlier run of this file) it's renamed
--      in place rather than dropped, so no data is lost.
--   3. api_attributes: the 'subTransaction' generic-API filter definition
--      pointed at main_transaction_id; repointed at order_id now that the
--      column is gone.
--
-- Run once, after schema.sql and migration_product_code.sql:
--   psql -U postgres -d kuttybrothers -f migration_transaction_invoice.sql
-- Idempotent — safe to run again on an already-migrated database (the app
-- does this itself on every `go run`; see pkg/database/migrate.go).
--

BEGIN;

ALTER TABLE public.transaction DROP CONSTRAINT IF EXISTS transaction_main_transaction_id_fkey;
DROP INDEX IF EXISTS idx_transaction_main_id;
ALTER TABLE public.transaction DROP COLUMN IF EXISTS main_transaction_id;

ALTER TABLE public.transaction ADD COLUMN IF NOT EXISTS invoice_id character varying;
ALTER TABLE public.transaction ADD COLUMN IF NOT EXISTS from_date  character varying(20);
ALTER TABLE public.transaction ADD COLUMN IF NOT EXISTS to_date    character varying(20);

-- invoice_type used to be added as tax_type; rename in place (preserving any
-- data already written) instead of dropping it, guarded so this is a no-op
-- both on a database that never had tax_type and on one already renamed.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'transaction' AND column_name = 'tax_type'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'transaction' AND column_name = 'invoice_type'
    ) THEN
        ALTER TABLE public.transaction RENAME COLUMN tax_type TO invoice_type;
    END IF;
END $$;

ALTER TABLE public.transaction ADD COLUMN IF NOT EXISTS invoice_type text;

CREATE INDEX IF NOT EXISTS idx_transaction_invoice_id ON public.transaction (invoice_id);
CREATE INDEX IF NOT EXISTS idx_transaction_order_id    ON public.transaction (order_id);

-- Repoint the generic-API filter definition at order_id; drop the stale one
-- referencing the column just removed above.
DELETE FROM public.api_attributes
WHERE api = 'subTransaction' AND fields->>'key' = 'main_transaction_id';

INSERT INTO public.api_attributes (api, type, fields)
SELECT v.api, v.type, v.fields::jsonb
FROM (VALUES
    ('subTransaction', 'api', '{"key": "order_id", "table": "transaction", "condition": "="}'),
    ('subTransaction', 'api', '{"key": "invoice_id", "table": "transaction", "condition": "="}')
) AS v(api, type, fields)
WHERE NOT EXISTS (
    SELECT 1 FROM public.api_attributes a
    WHERE a.api = v.api AND a.type = v.type AND a.fields = v.fields::jsonb
);

COMMIT;
