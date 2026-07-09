package treasury

import "errors"

var ErrInvalidAccountType = errors.New("invalid account type")

// Account is one node of the active chart of accounts. Codes are
// hierarchical; GFSM/COFOG mappings keep custom charts internationally
// comparable (design-language and blueprint §5.2).
type Account struct {
	Code        string
	Name        string
	AccountType string
	ParentCode  string
	GfsmCode    string
	CofogCode   string
	Active      bool
}

var ValidAccountTypes = map[string]struct{}{
	"ASSET":     {},
	"LIABILITY": {},
	"NET_WORTH": {},
	"REVENUE":   {},
	"EXPENSE":   {},
}

type ListAccountsFilter struct {
	AccountType string
	Pagination
}

type AccountPage struct {
	Accounts []Account
	Total    int
}
