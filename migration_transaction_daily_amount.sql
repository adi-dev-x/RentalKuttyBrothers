--
-- Kutty Brothers — Rental Software
-- Upgrade migration: transaction.daily_amount
--
-- Adds daily_amount, the per-day rate AddSubTransaction now multiplies by
-- the number of days between from_date and to_date to derive amount —
-- fixing amount being whatever raw total the client sent, disconnected from
-- the billing period. See model.TransactionDays and
-- repository.AddSubTransaction / GetSubTransactions.
--
-- Run once, after schema.sql and the other migration_*.sql files:
--   psql -U postgres -d kuttybrothers -f migration_transaction_daily_amount.sql
-- Idempotent — safe to run again on an already-migrated database (the app
-- does this itself on every `go run`; see pkg/database/migrate.go).
--

BEGIN;

ALTER TABLE public.transaction ADD COLUMN IF NOT EXISTS daily_amount numeric(12,2);

COMMIT;
