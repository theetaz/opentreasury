CREATE TABLE chart_of_accounts (
  version INTEGER PRIMARY KEY,
  status TEXT NOT NULL CHECK (status IN ('DRAFT', 'ACTIVE', 'RETIRED')),
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE coa_accounts (
  version INTEGER NOT NULL REFERENCES chart_of_accounts (version),
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  account_type TEXT NOT NULL
    CHECK (account_type IN ('ASSET', 'LIABILITY', 'NET_WORTH', 'REVENUE', 'EXPENSE')),
  parent_code TEXT,
  gfsm_code TEXT,
  cofog_code TEXT,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  PRIMARY KEY (version, code),
  FOREIGN KEY (version, parent_code) REFERENCES coa_accounts (version, code)
);

CREATE INDEX coa_accounts_account_type_idx
  ON coa_accounts (version, account_type);

-- Only one chart version may be ACTIVE at a time.
CREATE UNIQUE INDEX chart_of_accounts_single_active_idx
  ON chart_of_accounts (status)
  WHERE status = 'ACTIVE';
