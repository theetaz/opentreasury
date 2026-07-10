-- Demo institutions matching the web app's reference dataset.
INSERT INTO treasury_institutions (id, name, type, country_code, status) VALUES
  ('minfin', 'Ministry of Finance', 'MINISTRY', 'KE', 'ACTIVE'),
  ('health', 'Ministry of Health', 'MINISTRY', 'KE', 'ACTIVE'),
  ('transport', 'Transport Infrastructure Agency', 'AGENCY', 'KE', 'ACTIVE')
ON CONFLICT (id) DO NOTHING;
