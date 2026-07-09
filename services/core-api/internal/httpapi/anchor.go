package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type AnchorRepository interface {
	EntryProof(context.Context, string) (treasury.EntryProof, error)
}

func WithAnchorRepository(repository AnchorRepository) RouterOption {
	return func(config *routerConfig) {
		config.anchorRepository = repository
	}
}

type proofLineResponse struct {
	AccountCode string `json:"accountCode"`
	Direction   string `json:"direction"`
	AmountMinor int64  `json:"amountMinor"`
	Currency    string `json:"currency"`
}

type proofEntryResponse struct {
	ID            string              `json:"id"`
	InstitutionID string              `json:"institutionId"`
	FiscalYear    int                 `json:"fiscalYear"`
	EffectiveDate string              `json:"effectiveDate"`
	Status        string              `json:"status"`
	EntryType     string              `json:"entryType"`
	Lines         []proofLineResponse `json:"lines"`
}

type anchorResponse struct {
	ID         string `json:"id"`
	MerkleRoot string `json:"merkleRoot"`
	Backend    string `json:"backend"`
	BackendRef string `json:"backendRef"`
	AnchoredAt string `json:"anchoredAt"`
}

type proofResponse struct {
	Entry    proofEntryResponse `json:"entry"`
	Anchor   anchorResponse     `json:"anchor"`
	LeafHash string             `json:"leafHash"`
	Proof    json.RawMessage    `json:"proof"`
}

// entryProof serves the tamper-evidence bundle for a journal entry. The same
// handler backs the operator (/v1) and public (/public/v1) routes — the proof
// contains only canonical, public-safe fields, so it is safe to publish.
func (config routerConfig) entryProof(response http.ResponseWriter, request *http.Request) {
	if config.anchorRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "anchor repository is not configured")
		return
	}

	entryID := request.PathValue("id")
	proof, err := config.anchorRepository.EntryProof(request.Context(), entryID)
	if errors.Is(err, treasury.ErrNoAnchor) {
		writeError(response, http.StatusNotFound, "entry has no anchor yet")
		return
	}
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	lines := make([]proofLineResponse, 0, len(proof.Entry.Lines))
	for _, line := range proof.Entry.Lines {
		lines = append(lines, proofLineResponse{
			AccountCode: line.AccountCode, Direction: line.Direction,
			AmountMinor: line.AmountMinor, Currency: line.Currency,
		})
	}

	response.Header().Set("Content-Type", "application/json")
	if isPublicPath(request) {
		// Anchored data is immutable; cache it hard.
		response.Header().Set("Cache-Control", "public, max-age=300")
	}
	_ = json.NewEncoder(response).Encode(proofResponse{
		Entry: proofEntryResponse{
			ID:            proof.Entry.ID,
			InstitutionID: proof.Entry.InstitutionID,
			FiscalYear:    proof.Entry.FiscalYear,
			EffectiveDate: proof.Entry.EffectiveDate,
			Status:        proof.Entry.Status,
			EntryType:     proof.Entry.EntryType,
			Lines:         lines,
		},
		Anchor: anchorResponse{
			ID:         proof.AnchorID,
			MerkleRoot: proof.MerkleRoot,
			Backend:    proof.Backend,
			BackendRef: proof.BackendRef,
			AnchoredAt: proof.AnchoredAt,
		},
		LeafHash: proof.LeafHash,
		Proof:    proof.Proof,
	})
}
