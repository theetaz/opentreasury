package connector

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Watcher polls a drop directory for CSV files, maps each through the
// profile, delivers the records, and moves the file to processed/ (or
// failed/ when it cannot be read at all). Redelivery is safe: the staging
// API deduplicates by source hash.
type Watcher struct {
	dir       string
	profile   Profile
	deliverer *Deliverer
	logger    *slog.Logger
	interval  time.Duration
}

func NewWatcher(dir string, profile Profile, deliverer *Deliverer, logger *slog.Logger) *Watcher {
	return &Watcher{
		dir:       dir,
		profile:   profile,
		deliverer: deliverer,
		logger:    logger,
		interval:  2 * time.Second,
	}
}

func (w *Watcher) Run(ctx context.Context) error {
	for _, sub := range []string{"processed", "failed"} {
		if err := os.MkdirAll(filepath.Join(w.dir, sub), 0o750); err != nil {
			return err
		}
	}

	w.logger.Info("watching for csv drops", "dir", w.dir, "profile", w.profile.Name())

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			w.sweep(ctx)
		}
	}
}

func (w *Watcher) sweep(ctx context.Context) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		w.logger.Error("reading drop directory", "error", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".csv") {
			continue
		}
		w.processFile(ctx, filepath.Join(w.dir, entry.Name()))
	}
}

// ProcessFile maps and delivers one file; exported for tests.
func (w *Watcher) processFile(ctx context.Context, path string) {
	logger := w.logger.With("file", filepath.Base(path))

	file, err := os.Open(path) // #nosec G304 -- operator-controlled drop directory
	if err != nil {
		logger.Error("opening file", "error", err)
		return
	}

	result, err := MapCSV(w.profile, file)
	_ = file.Close()
	if err != nil {
		logger.Error("file rejected", "error", err)
		w.move(path, "failed")
		return
	}

	for _, skipped := range result.Skipped {
		logger.Warn("row skipped by connector", "row", skipped.RowNumber, "reason", skipped.Reason)
	}

	outcomes, err := w.deliverer.Deliver(ctx, result.Records)
	if err != nil {
		// Leave the file in place: the next sweep retries, and the staging
		// API deduplicates whatever was already delivered.
		logger.Error("delivery failed; will retry", "error", err, "delivered", len(outcomes))
		return
	}

	counts := map[string]int{}
	for _, outcome := range outcomes {
		counts[outcome.Status]++
		if outcome.Status == "QUARANTINED" {
			logger.Warn("record quarantined", "source_ref", outcome.SourceRef, "reason", outcome.Reason)
		}
	}

	logger.Info("file processed",
		"rows", len(result.Records),
		"posted", counts["POSTED"],
		"quarantined", counts["QUARANTINED"],
		"duplicate", counts["DUPLICATE"],
		"skipped", len(result.Skipped),
	)
	w.move(path, "processed")
}

func (w *Watcher) move(path, subdir string) {
	target := filepath.Join(w.dir, subdir, fmt.Sprintf("%s.%d", filepath.Base(path), time.Now().Unix()))
	if err := os.Rename(path, target); err != nil {
		w.logger.Error("moving file", "error", err)
	}
}
