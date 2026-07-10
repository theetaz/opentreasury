-- Demo transactions matching the web app's reference dataset.
INSERT INTO treasury_transactions
  (id, institution_id, fiscal_year, amount_minor, currency, description, transaction_date) VALUES
  ('txn-2026-0001', 'minfin', 2026, 125000, 'USD', 'Road maintenance payment', '2026-06-28'),
  ('txn-2026-0000', 'transport', 2026, 98000, 'USD', 'Bridge inspection payment', '2026-06-27'),
  ('txn-2025-0942', 'health', 2025, 450000, 'USD', 'Clinic equipment procurement', '2025-12-18')
ON CONFLICT (id) DO NOTHING;
