DROP INDEX IF EXISTS journal_entries_idempotency_key_idx;
DROP INDEX IF EXISTS journal_lines_account_code_idx;
DROP INDEX IF EXISTS journal_lines_entry_id_idx;
DROP INDEX IF EXISTS journal_entries_effective_date_idx;
DROP INDEX IF EXISTS journal_entries_institution_id_idx;
DROP TABLE IF EXISTS journal_lines;
DROP TABLE IF EXISTS journal_entries;
