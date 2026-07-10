// Package anchor batches posted journal entries, builds a Merkle tree over
// their canonical hashes, commits the root to a pluggable ledger backend, and
// records per-entry inclusion proofs.
package anchor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
)

// Backend is the tamper-evident ledger the Merkle root is committed to. The
// transparency-log implementation ships for local dev; a Fabric implementation
// satisfies the same interface in production (ADR-0004).
type Backend interface {
	// Name identifies the backend in anchor receipts.
	Name() string
	// Commit appends a Merkle root covering entryCount entries and returns a
	// backend reference (sequence or transaction id) proving it was recorded.
	Commit(ctx context.Context, merkleRoot string, entryCount int) (backendRef string, err error)
}

// TransparencyLog is an append-only, hash-chained log in Postgres: each row
// commits a Merkle root and chains the previous row's hash, so the sequence
// cannot be reordered or rewritten without detection.
type TransparencyLog struct {
	db *sql.DB
}

func NewTransparencyLog(db *sql.DB) *TransparencyLog {
	return &TransparencyLog{db: db}
}

func (*TransparencyLog) Name() string { return "transparency-log" }

func (l *TransparencyLog) Commit(ctx context.Context, merkleRoot string, _ int) (string, error) {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	var prevHash string
	err = tx.QueryRowContext(ctx,
		"SELECT entry_hash FROM transparency_log ORDER BY sequence DESC LIMIT 1",
	).Scan(&prevHash)
	if err == sql.ErrNoRows {
		prevHash = "genesis"
	} else if err != nil {
		return "", err
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(sequence), 0) + 1 FROM transparency_log",
	).Scan(&sequence); err != nil {
		return "", err
	}

	entryHash := chainHash(sequence, merkleRoot, prevHash)

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO transparency_log (sequence, merkle_root, prev_hash, entry_hash) VALUES ($1, $2, $3, $4)",
		sequence, merkleRoot, prevHash, entryHash,
	); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return fmt.Sprintf("tlog:%d", sequence), nil
}

func chainHash(sequence int64, merkleRoot, prevHash string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s", sequence, merkleRoot, prevHash)))
	return hex.EncodeToString(sum[:])
}
