-- Transactions gain a lifecycle status. Existing rows were all created by
-- the direct-post flow, so they backfill as POSTED via the default.
ALTER TABLE treasury_transactions
    ADD COLUMN status TEXT NOT NULL DEFAULT 'POSTED'
    CONSTRAINT treasury_transactions_status_check
    CHECK (status IN ('PENDING', 'POSTED', 'REJECTED'));

CREATE INDEX idx_treasury_transactions_status ON treasury_transactions (status);
