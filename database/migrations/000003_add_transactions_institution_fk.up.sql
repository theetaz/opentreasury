ALTER TABLE treasury_transactions
  ADD CONSTRAINT treasury_transactions_institution_id_fkey
  FOREIGN KEY (institution_id)
  REFERENCES treasury_institutions (id);
