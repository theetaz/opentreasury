CREATE TABLE treasury_transactions (
  id TEXT PRIMARY KEY,
  institution_id TEXT NOT NULL,
  fiscal_year INTEGER NOT NULL CHECK (fiscal_year > 0),
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  currency CHAR(3) NOT NULL,
  description TEXT NOT NULL,
  transaction_date DATE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX treasury_transactions_institution_id_idx
  ON treasury_transactions (institution_id);

CREATE INDEX treasury_transactions_transaction_date_idx
  ON treasury_transactions (transaction_date);
