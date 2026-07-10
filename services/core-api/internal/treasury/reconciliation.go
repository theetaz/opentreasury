package treasury

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// ReconciliationRow compares, for one source system and calendar month, what
// arrived in staging against what was actually posted to the journal. The
// staged amount counts only records that reached POSTED — quarantined records
// are surfaced by count so a human triages them, not silently summed.
type ReconciliationRow struct {
	SourceSystem      string
	Period            string // YYYY-MM
	StagedCount       int
	PostedCount       int
	QuarantinedCount  int
	StagedAmountMinor int64 // debit total of POSTED staged records
	PostedAmountMinor int64 // debit total of the journal entries they produced
}

type ListReconciliationFilter struct {
	SourceSystem string
	FiscalYear   int
	Pagination
}

type ReconciliationPage struct {
	Rows  []ReconciliationRow
	Total int
}

// PostgresReconciliationRepository aggregates staging records against the
// journal lines their POSTED records produced.
type PostgresReconciliationRepository struct {
	db *sql.DB
}

func NewPostgresReconciliationRepository(db *sql.DB) *PostgresReconciliationRepository {
	return &PostgresReconciliationRepository{db: db}
}

func (repository *PostgresReconciliationRepository) ListReconciliation(ctx context.Context, filter ListReconciliationFilter) (ReconciliationPage, error) {
	// Staged lines are the connector interchange JSON (Go field names); the
	// posted side reads the journal lines the staging pipeline created.
	query := `
		SELECT
			sr.source_system,
			to_char(date_trunc('month', sr.occurred_at), 'YYYY-MM') AS period,
			COUNT(*) AS staged_count,
			COUNT(*) FILTER (WHERE sr.status = 'POSTED') AS posted_count,
			COUNT(*) FILTER (WHERE sr.status = 'QUARANTINED') AS quarantined_count,
			COALESCE(SUM(staged.debit_minor) FILTER (WHERE sr.status = 'POSTED'), 0) AS staged_amount_minor,
			COALESCE(SUM(posted.debit_minor), 0) AS posted_amount_minor,
			COUNT(*) OVER () AS total_count
		FROM staging_records sr
		LEFT JOIN LATERAL (
			SELECT COALESCE(SUM((line->>'AmountMinor')::bigint), 0) AS debit_minor
			FROM jsonb_array_elements(sr.lines) AS line
			WHERE line->>'Direction' = 'DEBIT'
		) staged ON true
		LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(jl.amount_minor), 0) AS debit_minor
			FROM journal_lines jl
			WHERE jl.entry_id = sr.entry_id AND jl.direction = 'DEBIT'
		) posted ON true
	`

	args := make([]any, 0, 4)
	conditions := make([]string, 0, 2)

	if filter.SourceSystem != "" {
		args = append(args, filter.SourceSystem)
		conditions = append(conditions, fmt.Sprintf("sr.source_system = $%d", len(args)))
	}

	if filter.FiscalYear > 0 {
		args = append(args, filter.FiscalYear)
		conditions = append(conditions, fmt.Sprintf("EXTRACT(YEAR FROM sr.occurred_at) = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.PageSize, filter.Offset())
	query += fmt.Sprintf(`		GROUP BY sr.source_system, period
		ORDER BY period DESC, sr.source_system ASC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ReconciliationPage{}, err
	}
	defer func() { _ = rows.Close() }()

	page := ReconciliationPage{Rows: []ReconciliationRow{}}
	for rows.Next() {
		var row ReconciliationRow
		if err := rows.Scan(
			&row.SourceSystem,
			&row.Period,
			&row.StagedCount,
			&row.PostedCount,
			&row.QuarantinedCount,
			&row.StagedAmountMinor,
			&row.PostedAmountMinor,
			&page.Total,
		); err != nil {
			return ReconciliationPage{}, err
		}
		page.Rows = append(page.Rows, row)
	}

	return page, rows.Err()
}
