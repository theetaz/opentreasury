package treasury

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

type PostgresTransactionRepository struct {
	db *sql.DB
}

func NewPostgresTransactionRepository(db *sql.DB) *PostgresTransactionRepository {
	return &PostgresTransactionRepository{db: db}
}

func (repository *PostgresTransactionRepository) Save(ctx context.Context, tx Transaction) error {
	if err := ValidateTransaction(tx); err != nil {
		return err
	}

	_, err := repository.db.ExecContext(ctx, `
		INSERT INTO treasury_transactions (
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		tx.ID,
		tx.InstitutionID,
		tx.FiscalYear,
		tx.AmountMinor,
		tx.Currency,
		tx.Description,
		tx.TransactionDate,
	)
	return mapSaveError(tx.ID, err)
}

// mapSaveError converts constraint violations into domain errors the HTTP
// layer can translate into meaningful status codes.
func mapSaveError(transactionID string, err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return fmt.Errorf("saving transaction %q: %w", transactionID, ErrDuplicateTransaction)
		case pgForeignKeyViolation:
			return fmt.Errorf("saving transaction %q: %w", transactionID, ErrUnknownInstitution)
		}
	}

	return err
}

func (repository *PostgresTransactionRepository) List(ctx context.Context, filter ListTransactionsFilter) ([]Transaction, error) {
	query := `
		SELECT
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date::text
		FROM treasury_transactions
	`
	args := make([]any, 0, 3)
	conditions := make([]string, 0, 2)

	if filter.InstitutionID != "" {
		args = append(args, filter.InstitutionID)
		conditions = append(conditions, fmt.Sprintf("institution_id = $%d", len(args)))
	}

	if filter.FiscalYear > 0 {
		args = append(args, filter.FiscalYear)
		conditions = append(conditions, fmt.Sprintf("fiscal_year = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.Limit)
	query += fmt.Sprintf(`		ORDER BY transaction_date DESC, created_at DESC, id DESC
		LIMIT $%d
	`, len(args))

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var transactions []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(
			&tx.ID,
			&tx.InstitutionID,
			&tx.FiscalYear,
			&tx.AmountMinor,
			&tx.Currency,
			&tx.Description,
			&tx.TransactionDate,
		); err != nil {
			return nil, err
		}

		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (repository *PostgresTransactionRepository) ListAuditEvents(ctx context.Context, filter ListAuditEventsFilter) ([]AuditEvent, error) {
	query := `
		SELECT
			'audit-' || id || '-created' AS id,
			'TRANSACTION_CREATED' AS event_type,
			id AS transaction_id,
			institution_id,
			created_at::text AS occurred_at,
			'Transaction ' || id || ' was created.' AS summary
		FROM treasury_transactions
	`
	args := make([]any, 0, 2)
	conditions := make([]string, 0, 1)

	if filter.InstitutionID != "" {
		args = append(args, filter.InstitutionID)
		conditions = append(conditions, fmt.Sprintf("institution_id = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.Limit)
	query += fmt.Sprintf(`		ORDER BY created_at DESC, id DESC
		LIMIT $%d
	`, len(args))

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []AuditEvent
	for rows.Next() {
		var event AuditEvent
		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.TransactionID,
			&event.InstitutionID,
			&event.OccurredAt,
			&event.Summary,
		); err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}
