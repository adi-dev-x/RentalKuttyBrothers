--
-- Kutty Brothers — Rental Software
-- Upgrade migration: delivery_chelan.returned_at
--
-- Adds a returned_at timestamp to delivery_chelan so an order's actual
-- return date can be recorded directly, via POST /updateOrderReturnedAt.
-- delivery_items.returned_at already existed (POST /updateOrderItemReturnedAt
-- just needed the API, not a schema change); the order-level column did not.
--
-- Run once, after schema.sql and the other migration_*.sql files:
--   psql -U postgres -d kuttybrothers -f migration_order_returned_at.sql
-- Idempotent — safe to run again on an already-migrated database (the app
-- does this itself on every `go run`; see pkg/database/migrate.go).
--

BEGIN;

ALTER TABLE public.delivery_chelan ADD COLUMN IF NOT EXISTS returned_at timestamp without time zone;

CREATE INDEX IF NOT EXISTS idx_delivery_chelan_returned_at ON public.delivery_chelan (returned_at);

COMMIT;
