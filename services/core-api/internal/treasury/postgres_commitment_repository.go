package treasury

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// PostgresCommitmentRepository stores budget commitments and lists them with
// their settled totals.
type PostgresCommitmentRepository struct {
	db *sql.DB
}

func NewPostgresCommitmentRepository(db *sql.DB) *PostgresCommitmentRepository {
	return &PostgresCommitmentRepository{db: db}
}

func (repository *PostgresCommitmentRepository) CreateCommitment(ctx context.Context, commitment Commitment) error {
	if err := ValidateCommitment(commitment); err != nil {
		return err
	}

	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO commitments
			(id, institution_id, fiscal_year, account_code, description, amount_minor, currency, committed_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		commitment.ID,
		commitment.InstitutionID,
		commitment.FiscalYear,
		commitment.AccountCode,
		commitment.Description,
		commitment.AmountMinor,
		commitment.Currency,
		commitment.CommittedDate,
	)
	switch {
	case isUniqueViolation(err):
		return fmt.Errorf("creating commitment %q: %w", commitment.ID, ErrDuplicateCommitment)
	case isForeignKeyViolation(err):
		return fmt.Errorf("creating commitment %q: %w", commitment.ID, ErrUnknownInstitution)
	case err != nil:
		return err
	}

	metadata := auditMetadataFromContext(ctx)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_events (event_type, transaction_id, institution_id, summary, actor, request_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		"COMMITMENT_CREATED",
		commitment.ID,
		commitment.InstitutionID,
		"Commitment "+commitment.ID+" was created.",
		metadata.Actor,
		metadata.RequestID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (repository *PostgresCommitmentRepository) ListCommitments(ctx context.Context, filter ListCommitmentsFilter) (CommitmentPage, error) {
	query := `
		SELECT
			c.id,
			c.institution_id,
			c.fiscal_year,
			c.account_code,
			c.description,
			c.amount_minor,
			c.currency,
			c.committed_date::text,
			c.status,
			COALESCE(s.settled_minor, 0) AS settled_amount_minor,
			c.created_at::text,
			COUNT(*) OVER () AS total_count
		FROM commitments c
		LEFT JOIN LATERAL (
			SELECT SUM(cs.amount_minor) AS settled_minor
			FROM commitment_settlements cs
			WHERE cs.commitment_id = c.id
		) s ON true
	`

	args := make([]any, 0, 5)
	conditions := make([]string, 0, 3)

	if filter.InstitutionID != "" {
		args = append(args, filter.InstitutionID)
		conditions = append(conditions, fmt.Sprintf("c.institution_id = $%d", len(args)))
	}
	if filter.FiscalYear > 0 {
		args = append(args, filter.FiscalYear)
		conditions = append(conditions, fmt.Sprintf("c.fiscal_year = $%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.PageSize, filter.Offset())
	query += fmt.Sprintf(`		ORDER BY c.committed_date DESC, c.created_at DESC, c.id DESC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return CommitmentPage{}, err
	}
	defer func() { _ = rows.Close() }()

	page := CommitmentPage{Commitments: []Commitment{}}
	for rows.Next() {
		var commitment Commitment
		if err := rows.Scan(
			&commitment.ID,
			&commitment.InstitutionID,
			&commitment.FiscalYear,
			&commitment.AccountCode,
			&commitment.Description,
			&commitment.AmountMinor,
			&commitment.Currency,
			&commitment.CommittedDate,
			&commitment.Status,
			&commitment.SettledAmountMinor,
			&commitment.CreatedAt,
			&page.Total,
		); err != nil {
			return CommitmentPage{}, err
		}
		page.Commitments = append(page.Commitments, commitment)
	}

	return page, rows.Err()
}
