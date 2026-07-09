CREATE TABLE audit_events (
  id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
  event_type TEXT NOT NULL,
  transaction_id TEXT NOT NULL,
  institution_id TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  summary TEXT NOT NULL,
  actor TEXT NOT NULL DEFAULT 'system',
  request_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX audit_events_institution_id_idx
  ON audit_events (institution_id);

CREATE INDEX audit_events_occurred_at_idx
  ON audit_events (occurred_at);

-- Backfill creation events for transactions recorded before this table
-- existed, preserving their original timestamps.
INSERT INTO audit_events (id, event_type, transaction_id, institution_id, occurred_at, summary, actor)
SELECT
  'audit-' || id || '-created',
  'TRANSACTION_CREATED',
  id,
  institution_id,
  created_at,
  'Transaction ' || id || ' was created.',
  'system'
FROM treasury_transactions;
