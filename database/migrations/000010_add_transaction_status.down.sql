DROP INDEX IF EXISTS idx_treasury_transactions_status;
ALTER TABLE treasury_transactions DROP COLUMN IF EXISTS status;
