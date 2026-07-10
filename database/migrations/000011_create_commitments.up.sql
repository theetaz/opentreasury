-- Commitments encumber budget before money moves (blueprint 5.4). Journal
-- entries settle them through commitment_settlements, written in the same
-- transaction as the entry itself; settlements may never exceed the
-- committed amount (enforced in the posting transaction).
CREATE TABLE commitments (
  id TEXT PRIMARY KEY,
  institution_id TEXT NOT NULL REFERENCES treasury_institutions (id),
  fiscal_year INTEGER NOT NULL CHECK (fiscal_year > 0),
  account_code TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  committed_date DATE NOT NULL,
  status TEXT NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'SETTLED', 'CANCELLED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_commitments_institution ON commitments (institution_id);
CREATE INDEX idx_commitments_fiscal_year ON commitments (fiscal_year);
CREATE INDEX idx_commitments_status ON commitments (status);

CREATE TABLE commitment_settlements (
  commitment_id TEXT NOT NULL REFERENCES commitments (id),
  entry_id TEXT NOT NULL REFERENCES journal_entries (id),
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (commitment_id, entry_id)
);
