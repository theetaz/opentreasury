package treasury

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Commitment is a budget obligation recorded before money moves: a signed
// contract or purchase order that encumbers spending on an expense account.
// Journal entries settle it by referencing its id; settlements may never
// exceed the committed amount.
type Commitment struct {
	ID                 string
	InstitutionID      string
	FiscalYear         int
	AccountCode        string
	Description        string
	AmountMinor        int64
	Currency           string
	CommittedDate      string
	Status             string // OPEN, SETTLED, or CANCELLED
	SettledAmountMinor int64
	CreatedAt          string
}

var (
	ErrUnknownCommitment   = errors.New("commitment does not exist")
	ErrCommitmentClosed    = errors.New("commitment is not open")
	ErrCommitmentExceeded  = errors.New("settlement exceeds the committed amount")
	ErrCommitmentMismatch  = errors.New("entry institution or currency does not match the commitment")
	ErrDuplicateCommitment = errors.New("commitment id already exists")
)

var ValidCommitmentStatuses = map[string]struct{}{
	"OPEN":      {},
	"SETTLED":   {},
	"CANCELLED": {},
}

var commitmentCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

func ValidateCommitment(commitment Commitment) error {
	switch {
	case commitment.ID == "":
		return fmt.Errorf("commitment id is required")
	case commitment.InstitutionID == "":
		return fmt.Errorf("institutionId is required")
	case commitment.FiscalYear <= 0:
		return fmt.Errorf("fiscalYear must be positive")
	case commitment.AccountCode == "":
		return fmt.Errorf("accountCode is required")
	case commitment.AmountMinor <= 0:
		return fmt.Errorf("amountMinor must be positive")
	case !commitmentCurrencyPattern.MatchString(commitment.Currency):
		return fmt.Errorf("currency must be a three-letter ISO code")
	}

	if _, err := time.Parse(time.DateOnly, commitment.CommittedDate); err != nil {
		return fmt.Errorf("committedDate must be an ISO date: %w", err)
	}

	return nil
}

type ListCommitmentsFilter struct {
	InstitutionID string
	FiscalYear    int
	Status        string
	Pagination
}

type CommitmentPage struct {
	Commitments []Commitment
	Total       int
}
