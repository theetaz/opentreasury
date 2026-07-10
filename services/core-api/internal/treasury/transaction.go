package treasury

import (
	"errors"
	"time"
)

var (
	ErrMissingTransactionID   = errors.New("missing transaction id")
	ErrMissingInstitutionID   = errors.New("missing institution id")
	ErrMissingDescription     = errors.New("missing description")
	ErrMissingTransactionDate = errors.New("missing transaction date")
	ErrInvalidFiscalYear      = errors.New("invalid fiscal year")
	ErrInvalidAmount          = errors.New("invalid amount")
	ErrInvalidCurrency        = errors.New("invalid currency")
	ErrInvalidTransactionDate = errors.New("invalid transaction date")
	ErrInvalidLimit           = errors.New("invalid limit")
	ErrInvalidPage            = errors.New("invalid page")
	ErrInvalidPageSize        = errors.New("invalid page size")
	ErrInvalidDateRange       = errors.New("invalid date range")
	ErrInvalidAmountFilter    = errors.New("invalid amount filter")
	ErrInvalidStatus          = errors.New("invalid status")
	ErrDuplicateTransaction   = errors.New("transaction already exists")
	ErrUnknownInstitution     = errors.New("institution does not exist")
)

// Pagination is the common page window for all list queries.
// Page is 1-based; PageSize is capped by the HTTP layer.
type Pagination struct {
	Page     int
	PageSize int
}

func (p Pagination) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}

type Transaction struct {
	ID              string
	InstitutionID   string
	FiscalYear      int
	AmountMinor     int64
	Currency        string
	Description     string
	TransactionDate string
}

type ListTransactionsFilter struct {
	InstitutionID  string
	FiscalYear     int
	DateFrom       string // inclusive ISO date bound on transaction_date
	DateTo         string
	AmountMinorGte int64 // 0 means unset (amounts are strictly positive)
	AmountMinorLte int64
	Pagination
}

type TransactionPage struct {
	Transactions []Transaction
	Total        int
}

type AuditEvent struct {
	ID            string
	EventType     string
	TransactionID string
	InstitutionID string
	OccurredAt    string
	Summary       string
}

type ListAuditEventsFilter struct {
	InstitutionID string
	DateFrom      string // inclusive ISO date bound on the event timestamp
	DateTo        string
	Pagination
}

type AuditEventPage struct {
	Events []AuditEvent
	Total  int
}

type Institution struct {
	ID          string
	Name        string
	Type        string
	CountryCode string
	Status      string
}

type ListInstitutionsFilter struct {
	Status string
	Pagination
}

type InstitutionPage struct {
	Institutions []Institution
	Total        int
}

func ValidateTransaction(tx Transaction) error {
	switch {
	case tx.ID == "":
		return ErrMissingTransactionID
	case tx.InstitutionID == "":
		return ErrMissingInstitutionID
	case tx.Description == "":
		return ErrMissingDescription
	case tx.TransactionDate == "":
		return ErrMissingTransactionDate
	case !isISODate(tx.TransactionDate):
		return ErrInvalidTransactionDate
	case tx.FiscalYear <= 0:
		return ErrInvalidFiscalYear
	case tx.AmountMinor <= 0:
		return ErrInvalidAmount
	case !isISO4217CurrencyCode(tx.Currency):
		return ErrInvalidCurrency
	}

	return nil
}

func isISO4217CurrencyCode(value string) bool {
	if len(value) != 3 {
		return false
	}

	for _, char := range value {
		if char < 'A' || char > 'Z' {
			return false
		}
	}

	return true
}

func isISODate(value string) bool {
	_, err := time.Parse(time.DateOnly, value)
	return err == nil
}
