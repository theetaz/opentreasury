package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type PostgresStagingRepository struct {
	db      *sql.DB
	journal *PostgresJournalRepository
}

func NewPostgresStagingRepository(db *sql.DB, journal *PostgresJournalRepository) *PostgresStagingRepository {
	return &PostgresStagingRepository{db: db, journal: journal}
}

// Ingest processes one interchange record: redeliveries short-circuit on the
// source hash; valid records post a journal entry; invalid records are
// quarantined with the validation reason. The staging row records the outcome
// either way, so nothing submitted ever disappears.
func (repository *PostgresStagingRepository) Ingest(ctx context.Context, record StagingRecord) (StagingOutcome, error) {
	// At-least-once delivery: a known hash returns its original outcome.
	var status, reason, entryID string
	err := repository.db.QueryRowContext(ctx,
		"SELECT status, reason, COALESCE(entry_id, '') FROM staging_records WHERE source_hash = $1",
		record.SourceHash,
	).Scan(&status, &reason, &entryID)
	switch {
	case err == nil:
		return StagingOutcome{
			SourceRef:  record.SourceRef,
			SourceHash: record.SourceHash,
			Status:     "DUPLICATE",
			Reason:     reason,
			EntryID:    entryID,
		}, nil
	case !errors.Is(err, sql.ErrNoRows):
		return StagingOutcome{}, err
	}

	outcome := StagingOutcome{SourceRef: record.SourceRef, SourceHash: record.SourceHash}

	entry := record.ToJournalEntry()
	postErr := repository.journal.PostEntry(ctx, entry)
	switch {
	case postErr == nil:
		outcome.Status = "POSTED"
		outcome.EntryID = entry.ID
	case errors.Is(postErr, ErrDuplicateTransaction):
		// The entry exists (e.g. staging row write failed after a previous
		// post): treat as posted so redelivery converges.
		outcome.Status = "POSTED"
		outcome.EntryID = entry.ID
	default:
		outcome.Status = "QUARANTINED"
		outcome.Reason = postErr.Error()
	}

	linesJSON, err := json.Marshal(record.Lines)
	if err != nil {
		return StagingOutcome{}, err
	}

	var entryColumn any
	if outcome.EntryID != "" {
		entryColumn = outcome.EntryID
	}

	if _, err := repository.db.ExecContext(ctx, `
		INSERT INTO staging_records
			(source_system, source_ref, source_hash, profile, institution_id, occurred_at, lines, status, reason, entry_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (source_hash) DO NOTHING
	`,
		record.SourceSystem, record.SourceRef, record.SourceHash, record.Profile,
		record.InstitutionID, record.OccurredAt, linesJSON, outcome.Status, outcome.Reason, entryColumn,
	); err != nil {
		return StagingOutcome{}, err
	}

	return outcome, nil
}

func (repository *PostgresStagingRepository) ListStagingRecords(ctx context.Context, filter ListStagingRecordsFilter) (StagingRecordPage, error) {
	query := `
		SELECT
			id, source_system, source_ref, source_hash, profile,
			institution_id, occurred_at::text, lines, status, reason,
			COALESCE(entry_id, ''), created_at::text,
			COUNT(*) OVER() AS total_count
		FROM staging_records
	`
	args := make([]any, 0, 5)
	conditions := make([]string, 0, 3)

	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.InstitutionID != "" {
		args = append(args, filter.InstitutionID)
		conditions = append(conditions, fmt.Sprintf("institution_id = $%d", len(args)))
	}
	if filter.SourceSystem != "" {
		args = append(args, filter.SourceSystem)
		conditions = append(conditions, fmt.Sprintf("source_system = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.PageSize, filter.Offset())
	query += fmt.Sprintf(`		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return StagingRecordPage{}, err
	}
	defer func() { _ = rows.Close() }()

	page := StagingRecordPage{Records: []StagingRecord{}}
	for rows.Next() {
		var record StagingRecord
		var linesJSON []byte
		if err := rows.Scan(
			&record.ID, &record.SourceSystem, &record.SourceRef, &record.SourceHash, &record.Profile,
			&record.InstitutionID, &record.OccurredAt, &linesJSON, &record.Status, &record.Reason,
			&record.EntryID, &record.CreatedAt, &page.Total,
		); err != nil {
			return StagingRecordPage{}, err
		}
		if err := json.Unmarshal(linesJSON, &record.Lines); err != nil {
			return StagingRecordPage{}, err
		}
		page.Records = append(page.Records, record)
	}

	return page, rows.Err()
}
