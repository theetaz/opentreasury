// verify is the standalone proof verifier: given an entry id and a public API
// URL, it fetches the entry and its inclusion proof, recomputes the canonical
// hash and walks the Merkle path, and confirms the entry was anchored
// unmodified — trusting nothing but the math. A citizen can run this against
// any deployment to independently check a public fund movement.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/opentreasury/opentreasury/services/ledger-gateway/internal/merkle"
)

type proofResponse struct {
	Entry struct {
		ID            string `json:"id"`
		InstitutionID string `json:"institutionId"`
		FiscalYear    int    `json:"fiscalYear"`
		EffectiveDate string `json:"effectiveDate"`
		Status        string `json:"status"`
		EntryType     string `json:"entryType"`
		Lines         []struct {
			AccountCode string `json:"accountCode"`
			Direction   string `json:"direction"`
			AmountMinor int64  `json:"amountMinor"`
			Currency    string `json:"currency"`
		} `json:"lines"`
	} `json:"entry"`
	Anchor struct {
		ID         string `json:"id"`
		MerkleRoot string `json:"merkleRoot"`
		Backend    string `json:"backend"`
		BackendRef string `json:"backendRef"`
		AnchoredAt string `json:"anchoredAt"`
	} `json:"anchor"`
	Proof []merkle.ProofStep `json:"proof"`
	Leaf  string             `json:"leafHash"`
}

func main() {
	apiURL := flag.String("api", "http://localhost:8080", "public API base URL")
	entryID := flag.String("entry", "", "journal entry id to verify")
	flag.Parse()

	if *entryID == "" {
		fmt.Fprintln(os.Stderr, "usage: verify -entry <id> [-api <url>]")
		os.Exit(2)
	}

	proof, err := fetchProof(*apiURL, *entryID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Recompute the leaf hash from the entry the API returned — independently
	// of the leafHash the API also sent.
	entry := merkle.Entry{
		ID:            proof.Entry.ID,
		InstitutionID: proof.Entry.InstitutionID,
		FiscalYear:    proof.Entry.FiscalYear,
		EffectiveDate: proof.Entry.EffectiveDate,
		Status:        proof.Entry.Status,
		EntryType:     proof.Entry.EntryType,
	}
	for _, line := range proof.Entry.Lines {
		entry.Lines = append(entry.Lines, merkle.Line{
			AccountCode: line.AccountCode, Direction: line.Direction,
			AmountMinor: line.AmountMinor, Currency: line.Currency,
		})
	}

	recomputed := merkle.CanonicalHash(entry)
	if recomputed != proof.Leaf {
		fmt.Printf("✗ FAIL — the entry does not hash to the anchored leaf.\n")
		fmt.Printf("  entry hashes to: %s\n  anchored leaf:   %s\n", recomputed, proof.Leaf)
		os.Exit(1)
	}

	if !merkle.Verify(recomputed, proof.Proof, proof.Anchor.MerkleRoot) {
		fmt.Printf("✗ FAIL — the inclusion proof does not reach the anchored root.\n")
		os.Exit(1)
	}

	fmt.Printf("✓ VERIFIED — entry %s is anchored, unmodified.\n", proof.Entry.ID)
	fmt.Printf("  leaf hash:   %s\n", recomputed)
	fmt.Printf("  merkle root: %s\n", proof.Anchor.MerkleRoot)
	fmt.Printf("  backend:     %s (%s)\n", proof.Anchor.Backend, proof.Anchor.BackendRef)
	fmt.Printf("  anchored at: %s\n", proof.Anchor.AnchoredAt)
}

func fetchProof(apiURL, entryID string) (*proofResponse, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	url := fmt.Sprintf("%s/public/v1/entries/%s/proof", apiURL, entryID)

	response, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching proof: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("entry %s has no anchor yet (not found)", entryID)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proof endpoint returned %d: %s", response.StatusCode, string(body))
	}

	var proof proofResponse
	if err := json.Unmarshal(body, &proof); err != nil {
		return nil, fmt.Errorf("parsing proof: %w", err)
	}
	return &proof, nil
}
