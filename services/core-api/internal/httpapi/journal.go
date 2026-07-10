package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type JournalRepository interface {
	PostEntry(context.Context, treasury.JournalEntry) error
	ListEntries(context.Context, treasury.ListJournalEntriesFilter) (treasury.JournalEntryPage, error)
	ListBalances(context.Context, treasury.ListBalancesFilter) (treasury.BalancePage, error)
}

func WithJournalRepository(repository JournalRepository) RouterOption {
	return func(config *routerConfig) {
		config.journalRepository = repository
	}
}

type journalLinePayload struct {
	AccountCode string `json:"accountCode"`
	Direction   string `json:"direction"`
	AmountMinor int64  `json:"amountMinor"`
	Currency    string `json:"currency"`
}

type journalEntryPayload struct {
	ID             string               `json:"id"`
	InstitutionID  string               `json:"institutionId"`
	FiscalYear     int                  `json:"fiscalYear"`
	EffectiveDate  string               `json:"effectiveDate"`
	Description    string               `json:"description"`
	IdempotencyKey string               `json:"idempotencyKey,omitempty"`
	Lines          []journalLinePayload `json:"lines"`
}

func (p journalEntryPayload) toEntry() treasury.JournalEntry {
	lines := make([]treasury.JournalLine, 0, len(p.Lines))
	for _, line := range p.Lines {
		lines = append(lines, treasury.JournalLine{
			AccountCode: line.AccountCode,
			Direction:   line.Direction,
			AmountMinor: line.AmountMinor,
			Currency:    line.Currency,
		})
	}
	return treasury.JournalEntry{
		ID:             p.ID,
		InstitutionID:  p.InstitutionID,
		FiscalYear:     p.FiscalYear,
		EffectiveDate:  p.EffectiveDate,
		Description:    p.Description,
		IdempotencyKey: p.IdempotencyKey,
		Lines:          lines,
	}
}

type journalLineResponse struct {
	AccountCode string `json:"accountCode"`
	Direction   string `json:"direction"`
	AmountMinor int64  `json:"amountMinor"`
	Currency    string `json:"currency"`
}

type journalEntryResponse struct {
	ID              string                `json:"id"`
	InstitutionID   string                `json:"institutionId"`
	FiscalYear      int                   `json:"fiscalYear"`
	EffectiveDate   string                `json:"effectiveDate"`
	Description     string                `json:"description"`
	Status          string                `json:"status"`
	EntryType       string                `json:"entryType"`
	ReversesEntryID string                `json:"reversesEntryId,omitempty"`
	Lines           []journalLineResponse `json:"lines"`
}

type listJournalEntriesResponse struct {
	Entries    []journalEntryResponse `json:"entries"`
	Pagination paginationResponse     `json:"pagination"`
}

type balanceResponse struct {
	InstitutionID string `json:"institutionId"`
	AccountCode   string `json:"accountCode"`
	AccountName   string `json:"accountName,omitempty"`
	AccountType   string `json:"accountType,omitempty"`
	Currency      string `json:"currency"`
	BalanceMinor  int64  `json:"balanceMinor"`
}

type listBalancesResponse struct {
	Balances   []balanceResponse  `json:"balances"`
	Pagination paginationResponse `json:"pagination"`
}

func (config routerConfig) postJournalEntry(response http.ResponseWriter, request *http.Request) {
	if config.journalRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "journal repository is not configured")
		return
	}

	var payload journalEntryPayload
	if !decodeRequestBody(response, request, &payload) {
		return
	}

	entry := payload.toEntry()
	if err := treasury.ValidateJournalEntry(entry); err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	if !config.authorized(response, request, entry.InstitutionID) {
		return
	}

	ctx := treasury.ContextWithAuditMetadata(request.Context(), treasury.AuditMetadata{
		Actor:     "system",
		RequestID: requestIDFromContext(request.Context()),
	})

	if err := config.journalRepository.PostEntry(ctx, entry); err != nil {
		writeEntryError(response, request, config, err)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(response).Encode(map[string]string{"id": entry.ID})
}

func writeEntryError(response http.ResponseWriter, request *http.Request, config routerConfig, err error) {
	switch {
	case errors.Is(err, treasury.ErrDuplicateTransaction):
		writeError(response, http.StatusConflict, "journal entry already exists")
	case errors.Is(err, treasury.ErrUnknownInstitution):
		writeError(response, http.StatusUnprocessableEntity, treasury.ErrUnknownInstitution.Error())
	default:
		config.internalError(response, request, err)
	}
}

func (config routerConfig) listJournalEntries(response http.ResponseWriter, request *http.Request) {
	if !config.authorized(response, request, request.URL.Query().Get("institutionId")) {
		return
	}
	if config.journalRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "journal repository is not configured")
		return
	}

	filter, err := parseListJournalEntriesFilter(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	page, err := config.journalRepository.ListEntries(request.Context(), filter)
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	entries := make([]journalEntryResponse, 0, len(page.Entries))
	for _, entry := range page.Entries {
		entries = append(entries, toEntryResponse(entry))
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listJournalEntriesResponse{
		Entries:    entries,
		Pagination: toPaginationResponse(filter.Pagination, page.Total),
	})
}

func toEntryResponse(entry treasury.JournalEntry) journalEntryResponse {
	lines := make([]journalLineResponse, 0, len(entry.Lines))
	for _, line := range entry.Lines {
		lines = append(lines, journalLineResponse{
			AccountCode: line.AccountCode,
			Direction:   line.Direction,
			AmountMinor: line.AmountMinor,
			Currency:    line.Currency,
		})
	}
	return journalEntryResponse{
		ID:              entry.ID,
		InstitutionID:   entry.InstitutionID,
		FiscalYear:      entry.FiscalYear,
		EffectiveDate:   entry.EffectiveDate,
		Description:     entry.Description,
		Status:          entry.Status,
		EntryType:       entry.EntryType,
		ReversesEntryID: entry.ReversesEntryID,
		Lines:           lines,
	}
}

func (config routerConfig) listBalances(response http.ResponseWriter, request *http.Request) {
	if !config.authorized(response, request, request.URL.Query().Get("institutionId")) {
		return
	}
	if config.journalRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "balance repository is not configured")
		return
	}

	query := request.URL.Query()
	filter := treasury.ListBalancesFilter{
		InstitutionID: query.Get("institutionId"),
		AccountCode:   query.Get("accountCode"),
	}
	pagination, err := parsePagination(query)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}
	filter.Pagination = pagination

	page, err := config.journalRepository.ListBalances(request.Context(), filter)
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	balances := make([]balanceResponse, 0, len(page.Balances))
	for _, balance := range page.Balances {
		balances = append(balances, balanceResponse{
			InstitutionID: balance.InstitutionID,
			AccountCode:   balance.AccountCode,
			AccountName:   balance.AccountName,
			AccountType:   balance.AccountType,
			Currency:      balance.Currency,
			BalanceMinor:  balance.BalanceMinor,
		})
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listBalancesResponse{
		Balances:   balances,
		Pagination: toPaginationResponse(filter.Pagination, page.Total),
	})
}

var validEntryStatuses = map[string]struct{}{
	"POSTED":   {},
	"REVERSED": {},
}

func parseListJournalEntriesFilter(request *http.Request) (treasury.ListJournalEntriesFilter, error) {
	query := request.URL.Query()
	filter := treasury.ListJournalEntriesFilter{
		InstitutionID: query.Get("institutionId"),
	}

	if fiscalYear := query.Get("fiscalYear"); fiscalYear != "" {
		value, err := parsePositiveInt(fiscalYear)
		if err != nil {
			return treasury.ListJournalEntriesFilter{}, treasury.ErrInvalidFiscalYear
		}
		filter.FiscalYear = value
	}

	if status := query.Get("status"); status != "" {
		if _, ok := validEntryStatuses[status]; !ok {
			return treasury.ListJournalEntriesFilter{}, treasury.ErrInvalidStatus
		}
		filter.Status = status
	}

	var err error
	if filter.DateFrom, filter.DateTo, err = parseDateRange(query); err != nil {
		return treasury.ListJournalEntriesFilter{}, err
	}

	if filter.Pagination, err = parsePagination(query); err != nil {
		return treasury.ListJournalEntriesFilter{}, err
	}

	return filter, nil
}
