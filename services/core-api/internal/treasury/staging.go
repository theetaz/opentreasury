package treasury

// StagingRecord is one unit of the connector interchange format: a mapped
// source-system record awaiting posting. Delivery is at-least-once; the
// SourceHash deduplicates redeliveries end-to-end.
type StagingRecord struct {
	ID            string
	SourceSystem  string
	SourceRef     string
	SourceHash    string
	Profile       string
	InstitutionID string
	OccurredAt    string
	Lines         []JournalLine
	Status        string // POSTED or QUARANTINED
	Reason        string
	EntryID       string
	CreatedAt     string
}

// EntryID derives the deterministic journal entry id for a staging record, so
// redelivered records collide on the same entry rather than double-posting.
func (record StagingRecord) DeriveEntryID() string {
	return record.SourceSystem + "-" + record.SourceRef
}

// ToJournalEntry maps the staged record onto the double-entry model. The
// caller validates via PostEntry.
func (record StagingRecord) ToJournalEntry() JournalEntry {
	return JournalEntry{
		ID:             record.DeriveEntryID(),
		InstitutionID:  record.InstitutionID,
		FiscalYear:     fiscalYearOf(record.OccurredAt),
		EffectiveDate:  record.OccurredAt,
		Description:    "Imported from " + record.SourceSystem + " (" + record.SourceRef + ")",
		IdempotencyKey: record.SourceHash,
		Lines:          record.Lines,
	}
}

func fiscalYearOf(isoDate string) int {
	if len(isoDate) < 4 {
		return 0
	}
	year := 0
	for _, char := range isoDate[:4] {
		if char < '0' || char > '9' {
			return 0
		}
		year = year*10 + int(char-'0')
	}
	return year
}

type ListStagingRecordsFilter struct {
	Status        string
	InstitutionID string
	SourceSystem  string
	Pagination
}

type StagingRecordPage struct {
	Records []StagingRecord
	Total   int
}

// StagingOutcome reports what happened to one submitted record.
type StagingOutcome struct {
	SourceRef  string
	SourceHash string
	Status     string // POSTED, QUARANTINED, or DUPLICATE
	Reason     string
	EntryID    string
}
