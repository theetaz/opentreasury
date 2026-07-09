CREATE TABLE treasury_institutions (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  country_code CHAR(2) NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX treasury_institutions_status_idx
  ON treasury_institutions (status);

CREATE INDEX treasury_institutions_country_code_idx
  ON treasury_institutions (country_code);
