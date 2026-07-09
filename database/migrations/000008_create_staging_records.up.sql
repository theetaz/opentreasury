-- Staging records are the connector interchange landing zone: at-least-once
-- delivery deduplicated by source_hash, processed into journal entries or
-- quarantined with a machine-readable reason. Records are never deleted.
CREATE TABLE staging_records (
  id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
  source_system TEXT NOT NULL,
  source_ref TEXT NOT NULL,
  source_hash TEXT NOT NULL UNIQUE,
  profile TEXT NOT NULL,
  institution_id TEXT NOT NULL,
  occurred_at DATE NOT NULL,
  lines JSONB NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('POSTED', 'QUARANTINED')),
  reason TEXT NOT NULL DEFAULT '',
  entry_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX staging_records_status_idx ON staging_records (status);
CREATE INDEX staging_records_institution_id_idx ON staging_records (institution_id);
CREATE INDEX staging_records_source_system_idx ON staging_records (source_system);
