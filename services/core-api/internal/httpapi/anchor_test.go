package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/authz"
	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type stubAnchorRepository struct{ hasAnchor bool }

func (s stubAnchorRepository) EntryProof(_ context.Context, entryID string) (treasury.EntryProof, error) {
	if !s.hasAnchor {
		return treasury.EntryProof{}, treasury.ErrNoAnchor
	}
	return treasury.EntryProof{
		Entry: treasury.JournalEntry{
			ID: entryID, InstitutionID: "minfin", FiscalYear: 2026, EffectiveDate: "2026-07-01",
			Status: "POSTED", EntryType: "STANDARD",
			Lines: []treasury.JournalLine{{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 100, Currency: "USD"}},
		},
		AnchorID: "anc-1", MerkleRoot: "abc123", Backend: "transparency-log", BackendRef: "tlog:1",
		AnchoredAt: "2026-07-10T00:00:00Z", LeafHash: "leaf123",
		Proof: json.RawMessage(`[{"hash":"sib","left":false}]`),
	}, nil
}

func TestEntryProofReturnsBundle(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewRouter(WithAnchorRepository(stubAnchorRepository{hasAnchor: true})).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/entries/je-1/proof", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	var body proofResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "je-1", body.Entry.ID)
	require.Equal(t, "abc123", body.Anchor.MerkleRoot)
	require.Equal(t, "leaf123", body.LeafHash)
}

func TestEntryProofNotFoundWhenUnanchored(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewRouter(WithAnchorRepository(stubAnchorRepository{hasAnchor: false})).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/entries/je-x/proof", nil))

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestPublicEntryProofIsAnonymousAndCacheable(t *testing.T) {
	publication, err := authz.NewPublication(context.Background())
	require.NoError(t, err)
	router := NewRouter(
		WithAnchorRepository(stubAnchorRepository{hasAnchor: true}),
		WithPublication(publication),
		WithTokenVerifier(stubVerifier{ok: false}), // auth on: public path must bypass
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/public/v1/entries/je-1/proof", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Cache-Control"), "max-age")
}
