package treasury

import "errors"

var (
	ErrMissingEntryID       = errors.New("missing entry id")
	ErrMissingEntryInstID   = errors.New("missing institution id")
	ErrMissingEntryDesc     = errors.New("missing description")
	ErrInvalidEntryDate     = errors.New("invalid effective date")
	ErrInvalidEntryYear     = errors.New("invalid fiscal year")
	ErrInsufficientLines    = errors.New("a journal entry needs at least two lines")
	ErrInvalidLineAccount   = errors.New("missing account code")
	ErrInvalidLineDirection = errors.New("invalid line direction")
	ErrInvalidLineAmount    = errors.New("invalid line amount")
	ErrInvalidLineCurrency  = errors.New("invalid line currency")
	ErrUnbalancedEntry      = errors.New("entry debits and credits are not balanced per currency")
)

// JournalLine is one debit or credit leg posting to a chart-of-accounts code.
type JournalLine struct {
	LineNumber  int
	AccountCode string
	Direction   string // DEBIT or CREDIT
	AmountMinor int64
	Currency    string
}

// JournalEntry is a balanced double-entry event: the sum of debits equals the
// sum of credits within every currency. Entries are append-only once posted;
// a correction is a linked REVERSAL entry.
type JournalEntry struct {
	ID              string
	InstitutionID   string
	FiscalYear      int
	EffectiveDate   string
	Description     string
	Status          string // POSTED or REVERSED
	EntryType       string // STANDARD or REVERSAL
	ReversesEntryID string
	IdempotencyKey  string
	CommitmentID    string
	Lines           []JournalLine
}

func ValidateJournalEntry(entry JournalEntry) error {
	switch {
	case entry.ID == "":
		return ErrMissingEntryID
	case entry.InstitutionID == "":
		return ErrMissingEntryInstID
	case entry.Description == "":
		return ErrMissingEntryDesc
	case !isISODate(entry.EffectiveDate):
		return ErrInvalidEntryDate
	case entry.FiscalYear <= 0:
		return ErrInvalidEntryYear
	case len(entry.Lines) < 2:
		return ErrInsufficientLines
	}

	// Net amount per currency: debits add, credits subtract. A balanced entry
	// nets to zero in every currency it touches.
	netByCurrency := map[string]int64{}
	for _, line := range entry.Lines {
		if line.AccountCode == "" {
			return ErrInvalidLineAccount
		}
		if line.Direction != "DEBIT" && line.Direction != "CREDIT" {
			return ErrInvalidLineDirection
		}
		if line.AmountMinor <= 0 {
			return ErrInvalidLineAmount
		}
		if !isISO4217CurrencyCode(line.Currency) {
			return ErrInvalidLineCurrency
		}

		if line.Direction == "DEBIT" {
			netByCurrency[line.Currency] += line.AmountMinor
		} else {
			netByCurrency[line.Currency] -= line.AmountMinor
		}
	}

	for _, net := range netByCurrency {
		if net != 0 {
			return ErrUnbalancedEntry
		}
	}

	return nil
}

// Reversal produces the compensating entry that cancels this one: every line
// keeps its account, amount, and currency but flips direction.
func (entry JournalEntry) Reversal(id, effectiveDate string) JournalEntry {
	lines := make([]JournalLine, len(entry.Lines))
	for i, line := range entry.Lines {
		line.Direction = flipDirection(line.Direction)
		lines[i] = line
	}

	return JournalEntry{
		ID:              id,
		InstitutionID:   entry.InstitutionID,
		FiscalYear:      entry.FiscalYear,
		EffectiveDate:   effectiveDate,
		Description:     "Reversal of " + entry.ID,
		Status:          "POSTED",
		EntryType:       "REVERSAL",
		ReversesEntryID: entry.ID,
		Lines:           lines,
	}
}

func flipDirection(direction string) string {
	if direction == "DEBIT" {
		return "CREDIT"
	}
	return "DEBIT"
}

// SignedAmount is the balance contribution of a line: debits add, credits
// subtract. This is the single rule that keeps stocks consistent with flows.
func (line JournalLine) SignedAmount() int64 {
	if line.Direction == "DEBIT" {
		return line.AmountMinor
	}
	return -line.AmountMinor
}

type ListJournalEntriesFilter struct {
	InstitutionID string
	FiscalYear    int
	DateFrom      string
	DateTo        string
	Status        string
	Pagination
}

type JournalEntryPage struct {
	Entries []JournalEntry
	Total   int
}

type ListBalancesFilter struct {
	InstitutionID string
	AccountCode   string
	Pagination
}

// Balance is a materialized stock: the net posted position of one account for
// one institution in one currency.
type Balance struct {
	InstitutionID string
	AccountCode   string
	AccountName   string
	AccountType   string
	Currency      string
	BalanceMinor  int64
}

type BalancePage struct {
	Balances []Balance
	Total    int
}
