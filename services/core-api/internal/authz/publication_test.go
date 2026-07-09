package authz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func newPublication(t *testing.T) *Publication {
	t.Helper()
	publication, err := NewPublication(context.Background())
	require.NoError(t, err)
	return publication
}

func TestJournalDescriptionsAreNotPublished(t *testing.T) {
	publication := newPublication(t)

	// Descriptions can carry personal data; INSTITUTION-classified in v1.
	require.False(t, publication.PublishesField("journal-entries", "description"))
	require.False(t, publication.PublishesField("journal-entries", "idempotencyKey"))
	// Structure, amounts, and classification stay public.
	require.True(t, publication.PublishesField("journal-entries", "id"))
	require.True(t, publication.PublishesField("journal-entries", "lines"))
}

func TestRedactMasksUnlistedFields(t *testing.T) {
	publication := newPublication(t)

	masked := publication.Redact("journal-entries", map[string]any{
		"id":             "je-1",
		"institutionId":  "minfin",
		"description":    "Payment to J. Doe for consulting", // must never leak
		"idempotencyKey": "secret-key",
		"status":         "POSTED",
		"lines": []map[string]any{
			{"accountCode": "6202", "direction": "DEBIT", "amountMinor": int64(100), "currency": "USD", "internalNote": "x"},
		},
	})

	require.Equal(t, "je-1", masked["id"])
	require.NotContains(t, masked, "description")
	require.NotContains(t, masked, "idempotencyKey")

	lines := masked["lines"].([]map[string]any)
	require.Len(t, lines, 1)
	require.Equal(t, "6202", lines[0]["accountCode"])
	require.NotContains(t, lines[0], "internalNote")
}

func TestUnknownResourcePublishesNothing(t *testing.T) {
	publication := newPublication(t)

	masked := publication.Redact("secrets", map[string]any{"anything": "value"})
	require.Empty(t, masked)
}

func TestReferenceDataIsFullyPublic(t *testing.T) {
	publication := newPublication(t)

	for _, field := range []string{"code", "name", "accountType", "gfsmCode", "cofogCode", "depth"} {
		require.Truef(t, publication.PublishesField("accounts", field), "accounts.%s must be public", field)
	}
	for _, field := range []string{"institutionId", "accountCode", "balanceMinor", "currency"} {
		require.Truef(t, publication.PublishesField("balances", field), "balances.%s must be public", field)
	}
}
