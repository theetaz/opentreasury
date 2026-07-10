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

// Save records the transaction and its audit event in one database
// transaction: no domain change ever lands without its audit row.
func (repository *PostgresTransactionRepository) Save(ctx context.Context, tx Transaction) error {
	if err := ValidateTransaction(tx); err != nil {
		return err
	}

	dbTx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = dbTx.Rollback() }()

	_, err = dbTx.ExecContext(ctx, `
		INSERT INTO treasury_transactions (
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date,
			status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		tx.ID,
		tx.InstitutionID,
		tx.FiscalYear,
		tx.AmountMinor,
		tx.Currency,
		tx.Description,
		tx.TransactionDate,
		defaultStatus(tx.Status),
	)
	if err != nil {
		return mapSaveError(tx.ID, err)
	}

	metadata := auditMetadataFromContext(ctx)
	_, err = dbTx.ExecContext(ctx, `
		INSERT INTO audit_events (event_type, transaction_id, institution_id, summary, actor, request_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		"TRANSACTION_CREATED",
		tx.ID,
		tx.InstitutionID,
		"Transaction "+tx.ID+" was created.",
		metadata.Actor,
		metadata.RequestID,
	)
	if err != nil {
		return err
	}

	return dbTx.Commit()
}

// defaultStatus applies the write-path default: direct posts are POSTED.
func defaultStatus(status string) string {
	if status == "" {
		return "POSTED"
	}
	return status
}

// mapSaveError converts constraint violations into domain errors the HTTP
// layer can translate into meaningful status codes.
func mapSaveError(transactionID string, err error) error {
	switch {
	case err == nil:
		return nil
	case isUniqueViolation(err):
		return fmt.Errorf("saving transaction %q: %w", transactionID, ErrDuplicateTransaction)
	case isForeignKeyViolation(err):
		return fmt.Errorf("saving transaction %q: %w", transactionID, ErrUnknownInstitution)
	}

	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation
}

func (repository *PostgresTransactionRepository) List(ctx context.Context, filter ListTransactionsFilter) (TransactionPage, error) {
	query := `
		SELECT
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date::text,
			status,
			COUNT(*) OVER() AS total_count
		FROM treasury_transactions
	`
	args := make([]any, 0, 8)
	conditions := make([]string, 0, 6)

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
		conditions = append(conditions, fmt.Sprintf("transaction_date >= $%d", len(args)))
	}

	if filter.DateTo != "" {
		args = append(args, filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("transaction_date <= $%d", len(args)))
	}

	if filter.AmountMinorGte > 0 {
		args = append(args, filter.AmountMinorGte)
		conditions = append(conditions, fmt.Sprintf("amount_minor >= $%d", len(args)))
	}

	if filter.AmountMinorLte > 0 {
		args = append(args, filter.AmountMinorLte)
		conditions = append(conditions, fmt.Sprintf("amount_minor <= $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.PageSize, filter.Offset())
	query += fmt.Sprintf(`		ORDER BY transaction_date DESC, created_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return TransactionPage{}, err
	}
	defer func() { _ = rows.Close() }()

	page := TransactionPage{Transactions: []Transaction{}}
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
			&tx.Status,
			&page.Total,
		); err != nil {
			return TransactionPage{}, err
		}

		page.Transactions = append(page.Transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return TransactionPage{}, err
	}

	return page, nil
}

func (repository *PostgresTransactionRepository) ListAuditEvents(ctx context.Context, filter ListAuditEventsFilter) (AuditEventPage, error) {
	query := `
		SELECT
			id,
			event_type,
			transaction_id,
			institution_id,
			occurred_at::text AS occurred_at,
			summary,
			COUNT(*) OVER() AS total_count
		FROM audit_events
	`
	args := make([]any, 0, 5)
	conditions := make([]string, 0, 3)

	if filter.InstitutionID != "" {
		args = append(args, filter.InstitutionID)
		conditions = append(conditions, fmt.Sprintf("institution_id = $%d", len(args)))
	}

	if filter.DateFrom != "" {
		args = append(args, filter.DateFrom)
		conditions = append(conditions, fmt.Sprintf("occurred_at >= $%d::date", len(args)))
	}

	if filter.DateTo != "" {
		args = append(args, filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("occurred_at < ($%d::date + INTERVAL '1 day')", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.PageSize, filter.Offset())
	query += fmt.Sprintf(`		ORDER BY occurred_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return AuditEventPage{}, err
	}
	defer func() { _ = rows.Close() }()

	page := AuditEventPage{Events: []AuditEvent{}}
	for rows.Next() {
		var event AuditEvent
		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.TransactionID,
			&event.InstitutionID,
			&event.OccurredAt,
			&event.Summary,
			&page.Total,
		); err != nil {
			return AuditEventPage{}, err
		}

		page.Events = append(page.Events, event)
	}

	if err := rows.Err(); err != nil {
		return AuditEventPage{}, err
	}

	return page, nil
}
