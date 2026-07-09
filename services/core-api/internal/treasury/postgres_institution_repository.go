package treasury

import (
	"context"
	"database/sql"
)

type PostgresInstitutionRepository struct {
	db *sql.DB
}

func NewPostgresInstitutionRepository(db *sql.DB) *PostgresInstitutionRepository {
	return &PostgresInstitutionRepository{db: db}
}

func (repository *PostgresInstitutionRepository) ListInstitutions(ctx context.Context, filter ListInstitutionsFilter) ([]Institution, error) {
	query := `
		SELECT
			id,
			name,
			type,
			country_code,
			status
		FROM treasury_institutions
		ORDER BY name ASC, id ASC
		LIMIT $1
	`

	rows, err := repository.db.QueryContext(ctx, query, filter.Limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var institutions []Institution
	for rows.Next() {
		var institution Institution
		if err := rows.Scan(
			&institution.ID,
			&institution.Name,
			&institution.Type,
			&institution.CountryCode,
			&institution.Status,
		); err != nil {
			return nil, err
		}

		institutions = append(institutions, institution)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return institutions, nil
}
