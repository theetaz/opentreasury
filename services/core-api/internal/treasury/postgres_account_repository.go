package treasury

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type PostgresAccountRepository struct {
	db *sql.DB
}

func NewPostgresAccountRepository(db *sql.DB) *PostgresAccountRepository {
	return &PostgresAccountRepository{db: db}
}

// ListAccounts reads the ACTIVE chart version. Accounts are ordered by code,
// which walks the hierarchy in classification order.
func (repository *PostgresAccountRepository) ListAccounts(ctx context.Context, filter ListAccountsFilter) (AccountPage, error) {
	query := `
		SELECT
			a.code,
			a.name,
			a.account_type,
			COALESCE(a.parent_code, ''),
			COALESCE(a.gfsm_code, ''),
			COALESCE(a.cofog_code, ''),
			a.active,
			COUNT(*) OVER() AS total_count
		FROM coa_accounts a
		JOIN chart_of_accounts c ON c.version = a.version
		WHERE c.status = 'ACTIVE'
	`
	args := make([]any, 0, 3)

	if filter.AccountType != "" {
		args = append(args, filter.AccountType)
		query += fmt.Sprintf(" AND a.account_type = $%d\n", len(args))
	}

	args = append(args, filter.PageSize, filter.Offset())
	query += fmt.Sprintf(`		ORDER BY a.code ASC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := repository.db.QueryContext(ctx, strings.TrimSpace(query), args...)
	if err != nil {
		return AccountPage{}, err
	}
	defer func() { _ = rows.Close() }()

	page := AccountPage{Accounts: []Account{}}
	for rows.Next() {
		var account Account
		if err := rows.Scan(
			&account.Code,
			&account.Name,
			&account.AccountType,
			&account.ParentCode,
			&account.GfsmCode,
			&account.CofogCode,
			&account.Active,
			&page.Total,
		); err != nil {
			return AccountPage{}, err
		}

		page.Accounts = append(page.Accounts, account)
	}

	if err := rows.Err(); err != nil {
		return AccountPage{}, err
	}

	return page, nil
}
