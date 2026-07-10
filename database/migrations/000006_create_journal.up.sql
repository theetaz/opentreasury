CREATE TABLE journal_entries (
  id TEXT PRIMARY KEY,
  institution_id TEXT NOT NULL REFERENCES treasury_institutions (id),
  fiscal_year INTEGER NOT NULL CHECK (fiscal_year > 0),
  effective_date DATE NOT NULL,
  description TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('POSTED', 'REVERSED')),
  entry_type TEXT NOT NULL CHECK (entry_type IN ('STANDARD', 'REVERSAL')),
  reverses_entry_id TEXT REFERENCES journal_entries (id),
  idempotency_key TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE journal_lines (
  id BIGSERIAL PRIMARY KEY,
  entry_id TEXT NOT NULL REFERENCES journal_entries (id),
  line_number INTEGER NOT NULL,
  account_code TEXT NOT NULL,
  direction TEXT NOT NULL CHECK (direction IN ('DEBIT', 'CREDIT')),
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  currency CHAR(3) NOT NULL,
  UNIQUE (entry_id, line_number)
);

CREATE INDEX journal_entries_institution_id_idx
  ON journal_entries (institution_id);

CREATE INDEX journal_entries_effective_date_idx
  ON journal_entries (effective_date);

CREATE INDEX journal_lines_entry_id_idx
  ON journal_lines (entry_id);

CREATE INDEX journal_lines_account_code_idx
  ON journal_lines (account_code);

CREATE UNIQUE INDEX journal_entries_idempotency_key_idx
  ON journal_entries (idempotency_key)
  WHERE idempotency_key IS NOT NULL;
