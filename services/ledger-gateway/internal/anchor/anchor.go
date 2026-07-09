package anchor

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/opentreasury/opentreasury/services/ledger-gateway/internal/merkle"
)

// Service anchors posted journal entries into the backend ledger, in batches,
// on an interval. Anchoring is crash-safe: an entry is anchored exactly once
// because the write of its proof row is what marks it anchored.
type Service struct {
	db        *sql.DB
	backend   Backend
	logger    *slog.Logger
	batchSize int
	interval  time.Duration
}

func NewService(db *sql.DB, backend Backend, logger *slog.Logger) *Service {
	return &Service{
		db:        db,
		backend:   backend,
		logger:    logger,
		batchSize: 500,
		interval:  10 * time.Second,
	}
}

func (s *Service) Run(ctx context.Context) error {
	s.logger.Info("ledger gateway anchoring started", "backend", s.backend.Name(), "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Anchor immediately on start, then on each tick.
	s.anchorOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.anchorOnce(ctx)
		}
	}
}

func (s *Service) anchorOnce(ctx context.Context) {
	anchored, err := s.AnchorPending(ctx)
	if err != nil {
		s.logger.Error("anchoring batch failed", "error", err)
		return
	}
	if anchored > 0 {
		s.logger.Info("batch anchored", "entries", anchored, "backend", s.backend.Name())
	}
}

// unanchoredEntry is a posted entry with no anchor proof yet.
type unanchoredEntry struct {
	entry merkle.Entry
}

// AnchorPending anchors up to batchSize unanchored posted entries and returns
// how many were anchored. Exposed for tests and one-shot runs.
func (s *Service) AnchorPending(ctx context.Context) (int, error) {
	entries, err := s.loadUnanchored(ctx)
	if err != nil {
		return 0, err
	}
	if len(entries) == 0 {
		return 0, nil
	}

	leaves := make([]string, len(entries))
	for i, item := range entries {
		leaves[i] = merkle.CanonicalHash(item.entry)
	}

	tree := merkle.NewTree(leaves)
	root := tree.Root()

	backendRef, err := s.backend.Commit(ctx, root)
	if err != nil {
		return 0, fmt.Errorf("committing root to backend: %w", err)
	}

	anchorID, err := newID()
	if err != nil {
		return 0, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO anchors (id, merkle_root, entry_count, backend, backend_ref) VALUES ($1, $2, $3, $4, $5)",
		anchorID, root, len(entries), s.backend.Name(), backendRef,
	); err != nil {
		return 0, err
	}

	for i, item := range entries {
		proof, err := tree.Proof(i)
		if err != nil {
			return 0, err
		}
		proofJSON, err := json.Marshal(proof)
		if err != nil {
			return 0, err
		}
		// ON CONFLICT DO NOTHING makes re-anchoring after a crash idempotent:
		// an entry already proven is skipped.
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO journal_entry_anchors (entry_id, anchor_id, leaf_index, leaf_hash, proof)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (entry_id) DO NOTHING
		`, item.entry.ID, anchorID, i, leaves[i], proofJSON); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return len(entries), nil
}

func (s *Service) loadUnanchored(ctx context.Context) ([]unanchoredEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.institution_id, e.fiscal_year, e.effective_date::text, e.status, e.entry_type
		FROM journal_entries e
		LEFT JOIN journal_entry_anchors a ON a.entry_id = e.id
		WHERE a.entry_id IS NULL
		ORDER BY e.created_at ASC, e.id ASC
		LIMIT $1
	`, s.batchSize)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var entries []unanchoredEntry
	var ids []string
	byID := map[string]int{}
	for rows.Next() {
		var e merkle.Entry
		if err := rows.Scan(&e.ID, &e.InstitutionID, &e.FiscalYear, &e.EffectiveDate, &e.Status, &e.EntryType); err != nil {
			return nil, err
		}
		byID[e.ID] = len(entries)
		ids = append(ids, e.ID)
		entries = append(entries, unanchoredEntry{entry: e})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, nil
	}

	// Hydrate lines (needed for the canonical hash) in one query.
	lineRows, err := s.db.QueryContext(ctx, `
		SELECT entry_id, account_code, direction, amount_minor, currency
		FROM journal_lines
		WHERE entry_id = ANY($1)
		ORDER BY entry_id, line_number
	`, ids)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lineRows.Close() }()

	for lineRows.Next() {
		var entryID string
		var line merkle.Line
		if err := lineRows.Scan(&entryID, &line.AccountCode, &line.Direction, &line.AmountMinor, &line.Currency); err != nil {
			return nil, err
		}
		if idx, ok := byID[entryID]; ok {
			entries[idx].entry.Lines = append(entries[idx].entry.Lines, line)
		}
	}

	return entries, lineRows.Err()
}

func newID() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "anc-" + hex.EncodeToString(buffer), nil
}
