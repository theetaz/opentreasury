package treasury

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validCommitment() Commitment {
	return Commitment{
		ID:            "com-2026-0001",
		InstitutionID: "minfin",
		FiscalYear:    2026,
		AccountCode:   "22",
		Description:   "Road maintenance framework contract",
		AmountMinor:   500000,
		Currency:      "USD",
		CommittedDate: "2026-07-01",
	}
}

func TestValidateCommitmentAcceptsAWellFormedCommitment(t *testing.T) {
	require.NoError(t, ValidateCommitment(validCommitment()))
}

func TestValidateCommitmentRejectsMissingAndMalformedFields(t *testing.T) {
	cases := map[string]func(*Commitment){
		"missing id":          func(c *Commitment) { c.ID = "" },
		"missing institution": func(c *Commitment) { c.InstitutionID = "" },
		"zero fiscal year":    func(c *Commitment) { c.FiscalYear = 0 },
		"missing account":     func(c *Commitment) { c.AccountCode = "" },
		"non-positive amount": func(c *Commitment) { c.AmountMinor = 0 },
		"bad currency":        func(c *Commitment) { c.Currency = "usd" },
		"malformed date":      func(c *Commitment) { c.CommittedDate = "01/07/2026" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			commitment := validCommitment()
			mutate(&commitment)
			require.Error(t, ValidateCommitment(commitment))
		})
	}
}
