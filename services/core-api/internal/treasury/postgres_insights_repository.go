package treasury

import (
	"context"
	"database/sql"
)

// PostgresInsightsRepository reads the posted-flow history the insight
// calculations run on. Both queries are bounded: insights look at posted
// entries only, and observations cap at a window large enough for the
// statistics without unbounded scans.
type PostgresInsightsRepository struct {
	db *sql.DB
}

func NewPostgresInsightsRepository(db *sql.DB) *PostgresInsightsRepository {
	return &PostgresInsightsRepository{db: db}
}

func (repository *PostgresInsightsRepository) MonthlyFlows(ctx context.Context, institutionID string) ([]MonthlyFlow, error) {
	query := `
		SELECT
			to_char(date_trunc('month', e.effective_date), 'YYYY-MM') AS period,
			SUM(jl.amount_minor) AS total_minor
		FROM journal_entries e
		JOIN journal_lines jl ON jl.entry_id = e.id AND jl.direction = 'DEBIT'
		WHERE e.status = 'POSTED'
	`
	args := []any{}
	if institutionID != "" {
		args = append(args, institutionID)
		query += " AND e.institution_id = $1"
	}
	query += `
		GROUP BY period
		ORDER BY period ASC
	`

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	flows := []MonthlyFlow{}
	for rows.Next() {
		var flow MonthlyFlow
		if err := rows.Scan(&flow.Period, &flow.TotalMinor); err != nil {
			return nil, err
		}
		flows = append(flows, flow)
	}
	return flows, rows.Err()
}

const observationWindow = 5000

func (repository *PostgresInsightsRepository) FlowObservations(ctx context.Context, institutionID string) ([]FlowObservation, error) {
	query := `
		SELECT e.id, e.institution_id, jl.account_code, jl.amount_minor, e.effective_date::text
		FROM journal_lines jl
		JOIN journal_entries e ON e.id = jl.entry_id
		WHERE jl.direction = 'DEBIT' AND e.status = 'POSTED'
	`
	args := []any{}
	if institutionID != "" {
		args = append(args, institutionID)
		query += " AND e.institution_id = $1"
	}
	args = append(args, observationWindow)
	if len(args) == 1 {
		query += " ORDER BY e.effective_date DESC, e.id LIMIT $1"
	} else {
		query += " ORDER BY e.effective_date DESC, e.id LIMIT $2"
	}

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	observations := []FlowObservation{}
	for rows.Next() {
		var observation FlowObservation
		if err := rows.Scan(
			&observation.EntryID,
			&observation.InstitutionID,
			&observation.AccountCode,
			&observation.AmountMinor,
			&observation.EffectiveDate,
		); err != nil {
			return nil, err
		}
		observations = append(observations, observation)
	}
	return observations, rows.Err()
}
