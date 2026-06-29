package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type TransactionRepository interface {
	Save(context.Context, treasury.Transaction) error
	List(context.Context, treasury.ListTransactionsFilter) ([]treasury.Transaction, error)
}

type AuditEventRepository interface {
	ListAuditEvents(context.Context, treasury.ListAuditEventsFilter) ([]treasury.AuditEvent, error)
}

type InstitutionRepository interface {
	ListInstitutions(context.Context, treasury.ListInstitutionsFilter) ([]treasury.Institution, error)
}

type RouterOption func(*routerConfig)

type routerConfig struct {
	transactionRepository TransactionRepository
	auditEventRepository  AuditEventRepository
	institutionRepository InstitutionRepository
	allowedOrigins        map[string]struct{}
}

func WithTransactionRepository(repository TransactionRepository) RouterOption {
	return func(config *routerConfig) {
		config.transactionRepository = repository
		if auditRepository, ok := repository.(AuditEventRepository); ok {
			config.auditEventRepository = auditRepository
		}
	}
}

func WithAuditEventRepository(repository AuditEventRepository) RouterOption {
	return func(config *routerConfig) {
		config.auditEventRepository = repository
	}
}

func WithInstitutionRepository(repository InstitutionRepository) RouterOption {
	return func(config *routerConfig) {
		config.institutionRepository = repository
	}
}

func WithAllowedOrigins(origins []string) RouterOption {
	return func(config *routerConfig) {
		config.allowedOrigins = make(map[string]struct{}, len(origins))
		for _, origin := range origins {
			if origin != "" {
				config.allowedOrigins[origin] = struct{}{}
			}
		}
	}
}

func NewRouter(options ...RouterOption) http.Handler {
	config := routerConfig{}
	for _, option := range options {
		option(&config)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /v1/audit-events", config.listAuditEvents)
	mux.HandleFunc("GET /v1/institutions", config.listInstitutions)
	mux.HandleFunc("GET /v1/transactions", config.listTransactions)
	mux.HandleFunc("POST /v1/transactions", config.createTransaction)
	mux.HandleFunc("POST /v1/transactions/validate", validateTransaction)
	return config.withCORS(mux)
}

func (config routerConfig) withCORS(next http.Handler) http.Handler {
	if len(config.allowedOrigins) == 0 {
		return next
	}

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if _, ok := config.allowedOrigins[origin]; ok {
			response.Header().Set("Access-Control-Allow-Origin", origin)
			response.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			response.Header().Set("Vary", "Origin")
		}

		if request.Method == http.MethodOptions {
			response.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(response, request)
	})
}

func health(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(map[string]string{
		"status": "ok",
	})
}

func validateTransaction(response http.ResponseWriter, request *http.Request) {
	var tx treasury.Transaction
	if err := json.NewDecoder(request.Body).Decode(&tx); err != nil {
		writeError(response, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := treasury.ValidateTransaction(tx); err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(validationResponse{
		Status:          "VALID",
		Code:            "VALIDATION_SUCCESS",
		Message:         "Transaction is valid.",
		TransactionID:   tx.ID,
		InstitutionID:   tx.InstitutionID,
		FiscalYear:      tx.FiscalYear,
		AmountMinor:     tx.AmountMinor,
		Currency:        tx.Currency,
		TransactionDate: tx.TransactionDate,
	})
}

type validationResponse struct {
	Status          string `json:"status"`
	Code            string `json:"code"`
	Message         string `json:"message"`
	TransactionID   string `json:"transactionId"`
	InstitutionID   string `json:"institutionId"`
	FiscalYear      int    `json:"fiscalYear"`
	AmountMinor     int64  `json:"amountMinor"`
	Currency        string `json:"currency"`
	TransactionDate string `json:"transactionDate"`
}

func (config routerConfig) createTransaction(response http.ResponseWriter, request *http.Request) {
	var tx treasury.Transaction
	if err := json.NewDecoder(request.Body).Decode(&tx); err != nil {
		writeError(response, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := treasury.ValidateTransaction(tx); err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	if config.transactionRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "transaction repository is not configured")
		return
	}

	if err := config.transactionRepository.Save(request.Context(), tx); err != nil {
		writeError(response, http.StatusInternalServerError, err.Error())
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(response).Encode(map[string]string{
		"id": tx.ID,
	})
}

func (config routerConfig) listTransactions(response http.ResponseWriter, request *http.Request) {
	if config.transactionRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "transaction repository is not configured")
		return
	}

	filter, err := parseListTransactionsFilter(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	transactions, err := config.transactionRepository.List(request.Context(), filter)
	if err != nil {
		writeError(response, http.StatusInternalServerError, err.Error())
		return
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listTransactionsResponse{
		Transactions: toTransactionResponses(transactions),
	})
}

func (config routerConfig) listAuditEvents(response http.ResponseWriter, request *http.Request) {
	if config.auditEventRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "audit event repository is not configured")
		return
	}

	filter, err := parseListAuditEventsFilter(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	events, err := config.auditEventRepository.ListAuditEvents(request.Context(), filter)
	if err != nil {
		writeError(response, http.StatusInternalServerError, err.Error())
		return
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listAuditEventsResponse{
		Events: toAuditEventResponses(events),
	})
}

func (config routerConfig) listInstitutions(response http.ResponseWriter, request *http.Request) {
	if config.institutionRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "institution repository is not configured")
		return
	}

	filter, err := parseListInstitutionsFilter(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	institutions, err := config.institutionRepository.ListInstitutions(request.Context(), filter)
	if err != nil {
		writeError(response, http.StatusInternalServerError, err.Error())
		return
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listInstitutionsResponse{
		Institutions: toInstitutionResponses(institutions),
	})
}

type listTransactionsResponse struct {
	Transactions []transactionResponse `json:"transactions"`
}

type listAuditEventsResponse struct {
	Events []auditEventResponse `json:"events"`
}

type listInstitutionsResponse struct {
	Institutions []institutionResponse `json:"institutions"`
}

type transactionResponse struct {
	ID              string `json:"id"`
	InstitutionID   string `json:"institutionId"`
	FiscalYear      int    `json:"fiscalYear"`
	AmountMinor     int64  `json:"amountMinor"`
	Currency        string `json:"currency"`
	Description     string `json:"description"`
	TransactionDate string `json:"transactionDate"`
}

type auditEventResponse struct {
	ID            string `json:"id"`
	EventType     string `json:"eventType"`
	TransactionID string `json:"transactionId"`
	InstitutionID string `json:"institutionId"`
	OccurredAt    string `json:"occurredAt"`
	Summary       string `json:"summary"`
}

type institutionResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	CountryCode string `json:"countryCode"`
	Status      string `json:"status"`
}

func parseListTransactionsFilter(request *http.Request) (treasury.ListTransactionsFilter, error) {
	query := request.URL.Query()
	filter := treasury.ListTransactionsFilter{
		InstitutionID: query.Get("institutionId"),
		Limit:         50,
	}

	if fiscalYear := query.Get("fiscalYear"); fiscalYear != "" {
		value, err := strconv.Atoi(fiscalYear)
		if err != nil || value <= 0 {
			return treasury.ListTransactionsFilter{}, treasury.ErrInvalidFiscalYear
		}
		filter.FiscalYear = value
	}

	if limit := query.Get("limit"); limit != "" {
		value, err := strconv.Atoi(limit)
		if err != nil || value <= 0 {
			return treasury.ListTransactionsFilter{}, treasury.ErrInvalidAmount
		}
		if value > 100 {
			value = 100
		}
		filter.Limit = value
	}

	return filter, nil
}

func parseListAuditEventsFilter(request *http.Request) (treasury.ListAuditEventsFilter, error) {
	query := request.URL.Query()
	filter := treasury.ListAuditEventsFilter{
		InstitutionID: query.Get("institutionId"),
		Limit:         50,
	}

	if limit := query.Get("limit"); limit != "" {
		value, err := strconv.Atoi(limit)
		if err != nil || value <= 0 {
			return treasury.ListAuditEventsFilter{}, treasury.ErrInvalidAmount
		}
		if value > 100 {
			value = 100
		}
		filter.Limit = value
	}

	return filter, nil
}

func parseListInstitutionsFilter(request *http.Request) (treasury.ListInstitutionsFilter, error) {
	query := request.URL.Query()
	filter := treasury.ListInstitutionsFilter{
		Limit: 50,
	}

	if limit := query.Get("limit"); limit != "" {
		value, err := strconv.Atoi(limit)
		if err != nil || value <= 0 {
			return treasury.ListInstitutionsFilter{}, treasury.ErrInvalidAmount
		}
		if value > 100 {
			value = 100
		}
		filter.Limit = value
	}

	return filter, nil
}

func toTransactionResponses(transactions []treasury.Transaction) []transactionResponse {
	responses := make([]transactionResponse, 0, len(transactions))
	for _, tx := range transactions {
		responses = append(responses, transactionResponse{
			ID:              tx.ID,
			InstitutionID:   tx.InstitutionID,
			FiscalYear:      tx.FiscalYear,
			AmountMinor:     tx.AmountMinor,
			Currency:        tx.Currency,
			Description:     tx.Description,
			TransactionDate: tx.TransactionDate,
		})
	}

	return responses
}

func toAuditEventResponses(events []treasury.AuditEvent) []auditEventResponse {
	responses := make([]auditEventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, auditEventResponse{
			ID:            event.ID,
			EventType:     event.EventType,
			TransactionID: event.TransactionID,
			InstitutionID: event.InstitutionID,
			OccurredAt:    event.OccurredAt,
			Summary:       event.Summary,
		})
	}

	return responses
}

func toInstitutionResponses(institutions []treasury.Institution) []institutionResponse {
	responses := make([]institutionResponse, 0, len(institutions))
	for _, institution := range institutions {
		responses = append(responses, institutionResponse{
			ID:          institution.ID,
			Name:        institution.Name,
			Type:        institution.Type,
			CountryCode: institution.CountryCode,
			Status:      institution.Status,
		})
	}

	return responses
}

func writeError(response http.ResponseWriter, statusCode int, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)
	_ = json.NewEncoder(response).Encode(map[string]string{
		"error": message,
	})
}
