package treasury

import "testing"

func TestValidateTransactionAcceptsValidTransaction(t *testing.T) {
	tx := Transaction{
		ID:              "txn-2026-0001",
		InstitutionID:   "minfin",
		FiscalYear:      2026,
		AmountMinor:     125000,
		Currency:        "USD",
		Description:     "Road maintenance payment",
		TransactionDate: "2026-06-28",
	}

	if err := ValidateTransaction(tx); err != nil {
		t.Fatalf("ValidateTransaction() error = %v, want nil", err)
	}
}

func TestValidateTransactionRejectsMissingRequiredFields(t *testing.T) {
	base := validTransaction()

	tests := []struct {
		name   string
		mutate func(*Transaction)
	}{
		{
			name: "missing id",
			mutate: func(tx *Transaction) {
				tx.ID = ""
			},
		},
		{
			name: "missing institution id",
			mutate: func(tx *Transaction) {
				tx.InstitutionID = ""
			},
		},
		{
			name: "missing description",
			mutate: func(tx *Transaction) {
				tx.Description = ""
			},
		},
		{
			name: "missing transaction date",
			mutate: func(tx *Transaction) {
				tx.TransactionDate = ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := base
			tt.mutate(&tx)

			if err := ValidateTransaction(tx); err == nil {
				t.Fatal("ValidateTransaction() error = nil, want validation error")
			}
		})
	}
}

func TestValidateTransactionRejectsInvalidTreasuryValues(t *testing.T) {
	base := validTransaction()

	tests := []struct {
		name   string
		mutate func(*Transaction)
	}{
		{
			name: "zero fiscal year",
			mutate: func(tx *Transaction) {
				tx.FiscalYear = 0
			},
		},
		{
			name: "negative amount",
			mutate: func(tx *Transaction) {
				tx.AmountMinor = -1
			},
		},
		{
			name: "zero amount",
			mutate: func(tx *Transaction) {
				tx.AmountMinor = 0
			},
		},
		{
			name: "lowercase currency",
			mutate: func(tx *Transaction) {
				tx.Currency = "usd"
			},
		},
		{
			name: "short currency",
			mutate: func(tx *Transaction) {
				tx.Currency = "US"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := base
			tt.mutate(&tx)

			if err := ValidateTransaction(tx); err == nil {
				t.Fatal("ValidateTransaction() error = nil, want validation error")
			}
		})
	}
}

func TestValidateTransactionRejectsInvalidTransactionDate(t *testing.T) {
	tx := validTransaction()
	tx.TransactionDate = "06/28/2026"

	if err := ValidateTransaction(tx); err == nil {
		t.Fatal("ValidateTransaction() error = nil, want validation error")
	}
}

func validTransaction() Transaction {
	return Transaction{
		ID:              "txn-2026-0001",
		InstitutionID:   "minfin",
		FiscalYear:      2026,
		AmountMinor:     125000,
		Currency:        "USD",
		Description:     "Road maintenance payment",
		TransactionDate: "2026-06-28",
	}
}
