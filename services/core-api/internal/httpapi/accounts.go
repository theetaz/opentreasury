package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type AccountRepository interface {
	ListAccounts(context.Context, treasury.ListAccountsFilter) (treasury.AccountPage, error)
}

func WithAccountRepository(repository AccountRepository) RouterOption {
	return func(config *routerConfig) {
		config.accountRepository = repository
	}
}

type accountResponse struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	AccountType string `json:"accountType"`
	ParentCode  string `json:"parentCode,omitempty"`
	GfsmCode    string `json:"gfsmCode,omitempty"`
	CofogCode   string `json:"cofogCode,omitempty"`
	Active      bool   `json:"active"`
	Depth       int    `json:"depth"`
}

type listAccountsResponse struct {
	Accounts   []accountResponse  `json:"accounts"`
	Pagination paginationResponse `json:"pagination"`
}

func (config routerConfig) listAccounts(response http.ResponseWriter, request *http.Request) {
	if config.accountRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "account repository is not configured")
		return
	}

	filter, err := parseListAccountsFilter(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	page, err := config.accountRepository.ListAccounts(request.Context(), filter)
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	responses := make([]accountResponse, 0, len(page.Accounts))
	for _, account := range page.Accounts {
		responses = append(responses, accountResponse{
			Code:        account.Code,
			Name:        account.Name,
			AccountType: account.AccountType,
			ParentCode:  account.ParentCode,
			GfsmCode:    account.GfsmCode,
			CofogCode:   account.CofogCode,
			Active:      account.Active,
			Depth:       account.Depth,
		})
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listAccountsResponse{
		Accounts:   responses,
		Pagination: toPaginationResponse(filter.Pagination, page.Total),
	})
}

func parseListAccountsFilter(request *http.Request) (treasury.ListAccountsFilter, error) {
	query := request.URL.Query()
	filter := treasury.ListAccountsFilter{}

	if accountType := query.Get("accountType"); accountType != "" {
		if _, ok := treasury.ValidAccountTypes[accountType]; !ok {
			return treasury.ListAccountsFilter{}, treasury.ErrInvalidAccountType
		}
		filter.AccountType = accountType
	}

	var err error
	if filter.Pagination, err = parsePagination(query); err != nil {
		return treasury.ListAccountsFilter{}, err
	}

	return filter, nil
}
