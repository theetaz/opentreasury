package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// ErrNoAnchor means an entry has not been anchored yet (it exists but the
// ledger gateway has not batched it).
var ErrNoAnchor = errors.New("entry has no anchor yet")

// EntryProof is the tamper-evidence bundle for one journal entry: the public
// entry, its anchor receipt, its Merkle leaf hash, and the inclusion proof.
type EntryProof struct {
	Entry      JournalEntry
	AnchorID   string
	MerkleRoot string
	Backend    string
	BackendRef string
	AnchoredAt string
	LeafHash   string
	Proof      json.RawMessage
}

type PostgresAnchorRepository struct {
	db *sql.DB
}

func NewPostgresAnchorRepository(db *sql.DB) *PostgresAnchorRepository {
	return &PostgresAnchorRepository{db: db}
}

// EntryProof returns the proof bundle for an entry, or ErrNoAnchor if it has
// not been anchored. The entry fields are the same canonical, public-safe set
// the gateway hashed, so a verifier can recompute the leaf independently.
func (repository *PostgresAnchorRepository) EntryProof(ctx context.Context, entryID string) (EntryProof, error) {
	var proof EntryProof
	err := repository.db.QueryRowContext(ctx, `
		SELECT
			e.id, e.institution_id, e.fiscal_year, e.effective_date::text, e.status, e.entry_type,
			a.id, a.merkle_root, a.backend, a.backend_ref, a.anchored_at::text,
			jea.leaf_hash, jea.proof
		FROM journal_entry_anchors jea
		JOIN journal_entries e ON e.id = jea.entry_id
		JOIN anchors a ON a.id = jea.anchor_id
		WHERE jea.entry_id = $1
	`, entryID).Scan(
		&proof.Entry.ID, &proof.Entry.InstitutionID, &proof.Entry.FiscalYear,
		&proof.Entry.EffectiveDate, &proof.Entry.Status, &proof.Entry.EntryType,
		&proof.AnchorID, &proof.MerkleRoot, &proof.Backend, &proof.BackendRef, &proof.AnchoredAt,
		&proof.LeafHash, &proof.Proof,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return EntryProof{}, ErrNoAnchor
	}
	if err != nil {
		return EntryProof{}, err
	}

	lineRows, err := repository.db.QueryContext(ctx, `
		SELECT account_code, direction, amount_minor, currency
		FROM journal_lines WHERE entry_id = $1 ORDER BY line_number
	`, entryID)
	if err != nil {
		return EntryProof{}, err
	}
	defer func() { _ = lineRows.Close() }()

	for lineRows.Next() {
		var line JournalLine
		if err := lineRows.Scan(&line.AccountCode, &line.Direction, &line.AmountMinor, &line.Currency); err != nil {
			return EntryProof{}, err
		}
		proof.Entry.Lines = append(proof.Entry.Lines, line)
	}

	return proof, lineRows.Err()
}
