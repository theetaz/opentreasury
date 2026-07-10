//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

func TestCreatingATransactionWritesARealAuditEvent(t *testing.T) {
	dsn := startPostgres(t)
	migrator := newMigrator(t, dsn)
	db := openDB(t, dsn)

	require.NoError(t, migrator.Up())

	_, err := db.Exec(
		`INSERT INTO treasury_institutions (id, name, type, country_code, status)
		 VALUES ('minfin', 'Ministry of Finance', 'MINISTRY', 'KE', 'ACTIVE')`,
	)
	require.NoError(t, err)

	repository := treasury.NewPostgresTransactionRepository(db)
	ctx := treasury.ContextWithAuditMetadata(context.Background(), treasury.AuditMetadata{
		Actor:     "system",
		RequestID: "req-integration-1",
	})

	require.NoError(t, repository.Save(ctx, treasury.Transaction{
		ID:              "txn-audit-check",
		InstitutionID:   "minfin",
		FiscalYear:      2026,
		AmountMinor:     5000,
		Currency:        "USD",
		Description:     "audit integration check",
		TransactionDate: "2026-07-09",
	}))

	var eventType, actor, requestID, summary string
	err = db.QueryRow(
		`SELECT event_type, actor, request_id, summary
		   FROM audit_events WHERE transaction_id = 'txn-audit-check'`,
	).Scan(&eventType, &actor, &requestID, &summary)
	require.NoError(t, err)
	require.Equal(t, "TRANSACTION_CREATED", eventType)
	require.Equal(t, "system", actor)
	require.Equal(t, "req-integration-1", requestID)
	require.Contains(t, summary, "txn-audit-check")

	// The audit page reads the same table back through the repository.
	page, err := repository.ListAuditEvents(context.Background(), treasury.ListAuditEventsFilter{
		InstitutionID: "minfin",
		Pagination:    treasury.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Equal(t, "txn-audit-check", page.Events[0].TransactionID)
}
