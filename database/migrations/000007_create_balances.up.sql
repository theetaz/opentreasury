-- Materialized stocks: the running balance per institution, account, and
-- currency, updated transactionally as journal entries post. Balance is the
-- signed net of posted lines (debits add, credits subtract).
CREATE TABLE account_balances (
  institution_id TEXT NOT NULL REFERENCES treasury_institutions (id),
  account_code TEXT NOT NULL,
  currency CHAR(3) NOT NULL,
  balance_minor BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (institution_id, account_code, currency)
);

CREATE INDEX account_balances_account_code_idx
  ON account_balances (account_code);
