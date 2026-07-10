package treasury

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type PostgresInstitutionRepository struct {
	db *sql.DB
}

func NewPostgresInstitutionRepository(db *sql.DB) *PostgresInstitutionRepository {
	return &PostgresInstitutionRepository{db: db}
}

func (repository *PostgresInstitutionRepository) ListInstitutions(ctx context.Context, filter ListInstitutionsFilter) (InstitutionPage, error) {
	query := `
		SELECT
			id,
			name,
			type,
			country_code,
			status,
			COUNT(*) OVER() AS total_count
		FROM treasury_institutions
	`
	args := make([]any, 0, 3)
	conditions := make([]string, 0, 1)

	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.PageSize, filter.Offset())
	query += fmt.Sprintf(`		ORDER BY name ASC, id ASC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return InstitutionPage{}, err
	}
	defer func() { _ = rows.Close() }()

	page := InstitutionPage{Institutions: []Institution{}}
	for rows.Next() {
		var institution Institution
		if err := rows.Scan(
			&institution.ID,
			&institution.Name,
			&institution.Type,
			&institution.CountryCode,
			&institution.Status,
			&page.Total,
		); err != nil {
			return InstitutionPage{}, err
		}

		page.Institutions = append(page.Institutions, institution)
	}

	if err := rows.Err(); err != nil {
		return InstitutionPage{}, err
	}

	return page, nil
}
