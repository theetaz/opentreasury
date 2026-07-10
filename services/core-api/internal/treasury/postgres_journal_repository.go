package treasury

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var errIntegration = errors.New("integration failure") // used by tests

type PostgresJournalRepository struct {
	db *sql.DB
}

func NewPostgresJournalRepository(db *sql.DB) *PostgresJournalRepository {
	return &PostgresJournalRepository{db: db}
}

// PostEntry validates the entry is balanced, then writes the entry, its lines,
// the balance updates, and the audit event in one database transaction. Stocks
// stay consistent with flows because both are written together or not at all.
func (repository *PostgresJournalRepository) PostEntry(ctx context.Context, entry JournalEntry) error {
	if err := ValidateJournalEntry(entry); err != nil {
		return err
	}

	if entry.Status == "" {
		entry.Status = "POSTED"
	}
	if entry.EntryType == "" {
		entry.EntryType = "STANDARD"
	}

	dbTx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = dbTx.Rollback() }()

	var idempotencyKey any
	if entry.IdempotencyKey != "" {
		idempotencyKey = entry.IdempotencyKey
	}
	var reverses any
	if entry.ReversesEntryID != "" {
		reverses = entry.ReversesEntryID
	}

	if _, err := dbTx.ExecContext(ctx, `
		INSERT INTO journal_entries
			(id, institution_id, fiscal_year, effective_date, description, status, entry_type, reverses_entry_id, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		entry.ID, entry.InstitutionID, entry.FiscalYear, entry.EffectiveDate,
		entry.Description, entry.Status, entry.EntryType, reverses, idempotencyKey,
	); err != nil {
		return mapEntryError(entry.ID, err)
	}

	for i, line := range entry.Lines {
		if _, err := dbTx.ExecContext(ctx, `
			INSERT INTO journal_lines (entry_id, line_number, account_code, direction, amount_minor, currency)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
			entry.ID, i+1, line.AccountCode, line.Direction, line.AmountMinor, line.Currency,
		); err != nil {
			return err
		}

		if _, err := dbTx.ExecContext(ctx, `
			INSERT INTO account_balances (institution_id, account_code, currency, balance_minor)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (institution_id, account_code, currency)
			DO UPDATE SET balance_minor = account_balances.balance_minor + EXCLUDED.balance_minor,
			              updated_at = now()
		`,
			entry.InstitutionID, line.AccountCode, line.Currency, line.SignedAmount(),
		); err != nil {
			return err
		}
	}

	metadata := auditMetadataFromContext(ctx)
	if _, err := dbTx.ExecContext(ctx, `
		INSERT INTO audit_events (event_type, transaction_id, institution_id, summary, actor, request_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		eventTypeFor(entry.EntryType), entry.ID, entry.InstitutionID,
		summaryFor(entry), metadata.Actor, metadata.RequestID,
	); err != nil {
		return err
	}

	return dbTx.Commit()
}

func eventTypeFor(entryType string) string {
	if entryType == "REVERSAL" {
		return "ENTRY_REVERSED"
	}
	return "ENTRY_POSTED"
}

func summaryFor(entry JournalEntry) string {
	if entry.EntryType == "REVERSAL" {
		return "Journal entry " + entry.ID + " reversed " + entry.ReversesEntryID + "."
	}
	return "Journal entry " + entry.ID + " was posted."
}

func mapEntryError(entryID string, err error) error {
	if isUniqueViolation(err) {
		return fmt.Errorf("posting entry %q: %w", entryID, ErrDuplicateTransaction)
	}
	if isForeignKeyViolation(err) {
		return fmt.Errorf("posting entry %q: %w", entryID, ErrUnknownInstitution)
	}
	return err
}

func (repository *PostgresJournalRepository) ListEntries(ctx context.Context, filter ListJournalEntriesFilter) (JournalEntryPage, error) {
	query := `
		SELECT
			id, institution_id, fiscal_year, effective_date::text, description,
			status, entry_type, COUNT(*) OVER() AS total_count
		FROM journal_entries
	`
	args := make([]any, 0, 7)
	conditions := make([]string, 0, 5)

	if filter.InstitutionID != "" {
		args = append(args, filter.InstitutionID)
		conditions = append(conditions, fmt.Sprintf("institution_id = $%d", len(args)))
	}
	if filter.FiscalYear > 0 {
		args = append(args, filter.FiscalYear)
		conditions = append(conditions, fmt.Sprintf("fiscal_year = $%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.DateFrom != "" {
		args = append(args, filter.DateFrom)
		conditions = append(conditions, fmt.Sprintf("effective_date >= $%d", len(args)))
	}
	if filter.DateTo != "" {
		args = append(args, filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("effective_date <= $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.PageSize, filter.Offset())
	query += fmt.Sprintf(`		ORDER BY effective_date DESC, created_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return JournalEntryPage{}, err
	}
	defer func() { _ = rows.Close() }()

	page := JournalEntryPage{Entries: []JournalEntry{}}
	entryIDs := make([]any, 0)
	indexByID := map[string]int{}
	for rows.Next() {
		var entry JournalEntry
		if err := rows.Scan(
			&entry.ID, &entry.InstitutionID, &entry.FiscalYear, &entry.EffectiveDate,
			&entry.Description, &entry.Status, &entry.EntryType, &page.Total,
		); err != nil {
			return JournalEntryPage{}, err
		}
		indexByID[entry.ID] = len(page.Entries)
		entryIDs = append(entryIDs, entry.ID)
		page.Entries = append(page.Entries, entry)
	}
	if err := rows.Err(); err != nil {
		return JournalEntryPage{}, err
	}

	if err := repository.hydrateLines(ctx, page.Entries, entryIDs, indexByID); err != nil {
		return JournalEntryPage{}, err
	}

	return page, nil
}

// hydrateLines loads all lines for the page's entries in a single query and
// attaches them, avoiding an N+1 per entry.
func (repository *PostgresJournalRepository) hydrateLines(ctx context.Context, entries []JournalEntry, entryIDs []any, indexByID map[string]int) error {
	if len(entryIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(entryIDs))
	for i := range entryIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	query := `
		SELECT entry_id, account_code, direction, amount_minor, currency
		FROM journal_lines
		WHERE entry_id IN (` + strings.Join(placeholders, ", ") + `)
		ORDER BY entry_id, line_number
	`

	rows, err := repository.db.QueryContext(ctx, query, entryIDs...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var entryID string
		var line JournalLine
		if err := rows.Scan(&entryID, &line.AccountCode, &line.Direction, &line.AmountMinor, &line.Currency); err != nil {
			return err
		}
		if idx, ok := indexByID[entryID]; ok {
			entries[idx].Lines = append(entries[idx].Lines, line)
		}
	}

	return rows.Err()
}

func (repository *PostgresJournalRepository) ListBalances(ctx context.Context, filter ListBalancesFilter) (BalancePage, error) {
	query := `
		SELECT
			b.institution_id,
			b.account_code,
			COALESCE(a.name, ''),
			COALESCE(a.account_type, ''),
			b.currency,
			b.balance_minor,
			COUNT(*) OVER() AS total_count
		FROM account_balances b
		LEFT JOIN coa_accounts a
			ON a.code = b.account_code
			AND a.version = (SELECT version FROM chart_of_accounts WHERE status = 'ACTIVE')
	`
	args := make([]any, 0, 4)
	conditions := make([]string, 0, 2)

	if filter.InstitutionID != "" {
		args = append(args, filter.InstitutionID)
		conditions = append(conditions, fmt.Sprintf("b.institution_id = $%d", len(args)))
	}
	if filter.AccountCode != "" {
		args = append(args, filter.AccountCode)
		conditions = append(conditions, fmt.Sprintf("b.account_code = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.PageSize, filter.Offset())
	query += fmt.Sprintf(`		ORDER BY b.account_code ASC, b.currency ASC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return BalancePage{}, err
	}
	defer func() { _ = rows.Close() }()

	page := BalancePage{Balances: []Balance{}}
	for rows.Next() {
		var balance Balance
		if err := rows.Scan(
			&balance.InstitutionID, &balance.AccountCode, &balance.AccountName,
			&balance.AccountType, &balance.Currency, &balance.BalanceMinor, &page.Total,
		); err != nil {
			return BalancePage{}, err
		}
		page.Balances = append(page.Balances, balance)
	}

	return page, rows.Err()
}
