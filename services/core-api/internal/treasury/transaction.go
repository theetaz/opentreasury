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
)

type Transaction struct {
	ID              string
	InstitutionID   string
	FiscalYear      int
	AmountMinor     int64
	Currency        string
	Description     string
	TransactionDate string
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
