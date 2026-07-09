package treasury

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func balancedEntry() JournalEntry {
	return JournalEntry{
		ID:            "je-2026-0001",
		InstitutionID: "minfin",
		FiscalYear:    2026,
		EffectiveDate: "2026-06-28",
		Description:   "Tax receipt into the treasury account",
		Lines: []JournalLine{
			{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 125000, Currency: "USD"},
			{AccountCode: "114", Direction: "CREDIT", AmountMinor: 125000, Currency: "USD"},
		},
	}
}

func TestValidateJournalEntryAcceptsABalancedEntry(t *testing.T) {
	require.NoError(t, ValidateJournalEntry(balancedEntry()))
}

func TestValidateJournalEntryRejectsUnbalancedEntry(t *testing.T) {
	entry := balancedEntry()
	entry.Lines[1].AmountMinor = 100000

	require.ErrorIs(t, ValidateJournalEntry(entry), ErrUnbalancedEntry)
}

func TestValidateJournalEntryBalancesPerCurrency(t *testing.T) {
	entry := balancedEntry()
	entry.Lines = []JournalLine{
		{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 100, Currency: "USD"},
		{AccountCode: "114", Direction: "CREDIT", AmountMinor: 100, Currency: "USD"},
		// EUR debit with no matching EUR credit — unbalanced in EUR.
		{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 50, Currency: "EUR"},
		{AccountCode: "114", Direction: "CREDIT", AmountMinor: 40, Currency: "EUR"},
	}

	require.ErrorIs(t, ValidateJournalEntry(entry), ErrUnbalancedEntry)
}

func TestValidateJournalEntryRequiresAtLeastTwoLines(t *testing.T) {
	entry := balancedEntry()
	entry.Lines = entry.Lines[:1]

	require.ErrorIs(t, ValidateJournalEntry(entry), ErrInsufficientLines)
}

func TestValidateJournalEntryRejectsInvalidLineFields(t *testing.T) {
	cases := map[string]func(*JournalEntry){
		"missing account": func(e *JournalEntry) { e.Lines[0].AccountCode = "" },
		"bad direction":   func(e *JournalEntry) { e.Lines[0].Direction = "SIDEWAYS" },
		"zero amount":     func(e *JournalEntry) { e.Lines[0].AmountMinor = 0; e.Lines[1].AmountMinor = 0 },
		"bad currency":    func(e *JournalEntry) { e.Lines[0].Currency = "US"; e.Lines[1].Currency = "US" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			entry := balancedEntry()
			mutate(&entry)
			require.Error(t, ValidateJournalEntry(entry))
		})
	}
}

func TestValidateJournalEntryRejectsMissingHeaderFields(t *testing.T) {
	cases := map[string]func(*JournalEntry){
		"missing id":          func(e *JournalEntry) { e.ID = "" },
		"missing institution": func(e *JournalEntry) { e.InstitutionID = "" },
		"missing description": func(e *JournalEntry) { e.Description = "" },
		"bad date":            func(e *JournalEntry) { e.EffectiveDate = "28-06-2026" },
		"bad fiscal year":     func(e *JournalEntry) { e.FiscalYear = 0 },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			entry := balancedEntry()
			mutate(&entry)
			require.Error(t, ValidateJournalEntry(entry))
		})
	}
}

func TestReversalEntryFlipsEveryLine(t *testing.T) {
	original := balancedEntry()
	original.Status = "POSTED"

	reversal := original.Reversal("je-rev-0001", "2026-07-01")

	require.Equal(t, "je-rev-0001", reversal.ID)
	require.Equal(t, "REVERSAL", reversal.EntryType)
	require.Equal(t, original.ID, reversal.ReversesEntryID)
	require.Len(t, reversal.Lines, len(original.Lines))
	require.Equal(t, "CREDIT", reversal.Lines[0].Direction) // was DEBIT
	require.Equal(t, "DEBIT", reversal.Lines[1].Direction)  // was CREDIT
	require.Equal(t, original.Lines[0].AmountMinor, reversal.Lines[0].AmountMinor)
	require.NoError(t, ValidateJournalEntry(reversal))
}
