package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/opentreasury/opentreasury/services/core-api/internal/auth"
	"github.com/opentreasury/opentreasury/services/core-api/internal/authz"
	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

const defaultMaxBodyBytes int64 = 1 << 20 // 1 MiB

const readinessCheckTimeout = 2 * time.Second

type TransactionRepository interface {
	Save(context.Context, treasury.Transaction) error
	List(context.Context, treasury.ListTransactionsFilter) (treasury.TransactionPage, error)
}

type AuditEventRepository interface {
	ListAuditEvents(context.Context, treasury.ListAuditEventsFilter) (treasury.AuditEventPage, error)
}

type InstitutionRepository interface {
	ListInstitutions(context.Context, treasury.ListInstitutionsFilter) (treasury.InstitutionPage, error)
}

type RouterOption func(*routerConfig)

type routerConfig struct {
	transactionRepository TransactionRepository
	auditEventRepository  AuditEventRepository
	institutionRepository InstitutionRepository
	accountRepository     AccountRepository
	journalRepository     JournalRepository
	tokenVerifier         auth.TokenVerifier
	authorizer            Authorizer
	publication           *authz.Publication
	publicRateRPS         float64
	publicRateBurst       int
	allowedOrigins        map[string]struct{}
	logger                *slog.Logger
	readinessChecks       []func(context.Context) error
	maxBodyBytes          int64
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

// WithLogger sets the structured logger used for request, error, and panic
// logging. Defaults to slog.Default().
func WithLogger(logger *slog.Logger) RouterOption {
	return func(config *routerConfig) {
		config.logger = logger
	}
}

// WithReadinessCheck registers a dependency probe consulted by GET /readyz.
// Any failing check makes readiness report 503.
func WithReadinessCheck(check func(context.Context) error) RouterOption {
	return func(config *routerConfig) {
		config.readinessChecks = append(config.readinessChecks, check)
	}
}

// WithMaxBodyBytes caps request body size; larger bodies are rejected with 413.
func WithMaxBodyBytes(limit int64) RouterOption {
	return func(config *routerConfig) {
		config.maxBodyBytes = limit
	}
}

// WithTokenVerifier turns on bearer-token authentication. When set, every
// route except health/readiness requires a valid token; when nil (the
// default), the API runs unauthenticated for local development.
func WithTokenVerifier(verifier auth.TokenVerifier) RouterOption {
	return func(config *routerConfig) {
		config.tokenVerifier = verifier
	}
}

// WithAuthorizer turns on policy-based authorization. When set, each request
// is checked against the policy (role × action × resource × institution scope)
// after authentication.
func WithAuthorizer(authorizer Authorizer) RouterOption {
	return func(config *routerConfig) {
		config.authorizer = authorizer
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
	config := routerConfig{
		logger:       slog.Default(),
		maxBodyBytes: defaultMaxBodyBytes,
	}
	for _, option := range options {
		option(&config)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /readyz", config.readiness)
	mux.HandleFunc("GET /v1/accounts", config.listAccounts)
	mux.HandleFunc("GET /v1/balances", config.listBalances)
	mux.HandleFunc("POST /v1/journal-entries", config.postJournalEntry)
	mux.HandleFunc("GET /v1/journal-entries", config.listJournalEntries)
	mux.HandleFunc("GET /v1/audit-events", config.listAuditEvents)
	mux.HandleFunc("GET /v1/institutions", config.listInstitutions)
	mux.HandleFunc("GET /v1/transactions", config.listTransactions)
	mux.HandleFunc("POST /v1/transactions", config.createTransaction)
	mux.HandleFunc("POST /v1/transactions/validate", validateTransaction)
	config.registerPublicRoutes(mux)

	var handler http.Handler = mux
	if config.tokenVerifier != nil {
		handler = auth.Middleware(config.tokenVerifier, isPublicPath, handler)
	}
	handler = config.withCORS(handler)
	handler = withMaxBodyBytes(config.maxBodyBytes, handler)
	handler = withRecovery(config.logger, handler)
	handler = withRequestLogging(config.logger, handler)
	handler = withRequestID(handler)
	return handler
}

// isPublicPath matches routes served without authentication: liveness/
// readiness probes and the anonymous public read tier.
func isPublicPath(request *http.Request) bool {
	path := request.URL.Path
	if path == "/healthz" || path == "/readyz" {
		return true
	}
	return strings.HasPrefix(path, "/public/")
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
			response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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

func (config routerConfig) readiness(response http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), readinessCheckTimeout)
	defer cancel()

	for _, check := range config.readinessChecks {
		if err := check(ctx); err != nil {
			config.logger.Error("readiness check failed",
				"request_id", requestIDFromContext(request.Context()),
				"error", err,
			)
			response.Header().Set("Content-Type", "application/json")
			response.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(response).Encode(map[string]string{"status": "unavailable"})
			return
		}
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(map[string]string{"status": "ready"})
}

// decodeRequestBody decodes JSON, mapping oversized bodies to 413 and other
// decode failures to 400. Returns false if a response was already written.
func decodeRequestBody(response http.ResponseWriter, request *http.Request, target any) bool {
	if err := json.NewDecoder(request.Body).Decode(target); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(response, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}

		writeError(response, http.StatusBadRequest, "invalid JSON")
		return false
	}

	return true
}

// internalError logs the real error for operators and returns a sanitized
// message to the client. Internal details must never reach response bodies.
func (config routerConfig) internalError(response http.ResponseWriter, request *http.Request, err error) {
	config.logger.Error("internal error",
		"request_id", requestIDFromContext(request.Context()),
		"method", request.Method,
		"path", request.URL.Path,
		"error", err,
	)
	writeError(response, http.StatusInternalServerError, "internal server error")
}

func validateTransaction(response http.ResponseWriter, request *http.Request) {
	var tx treasury.Transaction
	if !decodeRequestBody(response, request, &tx) {
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
	if !decodeRequestBody(response, request, &tx) {
		return
	}

	if err := treasury.ValidateTransaction(tx); err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	if !config.authorized(response, request, tx.InstitutionID) {
		return
	}

	if config.transactionRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "transaction repository is not configured")
		return
	}

	auditCtx := treasury.ContextWithAuditMetadata(request.Context(), treasury.AuditMetadata{
		Actor:     "system", // becomes the authenticated principal once identity lands (Phase 2)
		RequestID: requestIDFromContext(request.Context()),
	})

	if err := config.transactionRepository.Save(auditCtx, tx); err != nil {
		switch {
		case errors.Is(err, treasury.ErrDuplicateTransaction):
			writeError(response, http.StatusConflict, treasury.ErrDuplicateTransaction.Error())
		case errors.Is(err, treasury.ErrUnknownInstitution):
			writeError(response, http.StatusUnprocessableEntity, treasury.ErrUnknownInstitution.Error())
		default:
			config.internalError(response, request, err)
		}
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(response).Encode(map[string]string{
		"id": tx.ID,
	})
}

func (config routerConfig) listTransactions(response http.ResponseWriter, request *http.Request) {
	if !config.authorized(response, request, request.URL.Query().Get("institutionId")) {
		return
	}
	if config.transactionRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "transaction repository is not configured")
		return
	}

	filter, err := parseListTransactionsFilter(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	page, err := config.transactionRepository.List(request.Context(), filter)
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listTransactionsResponse{
		Transactions: toTransactionResponses(page.Transactions),
		Pagination:   toPaginationResponse(filter.Pagination, page.Total),
	})
}

func (config routerConfig) listAuditEvents(response http.ResponseWriter, request *http.Request) {
	if !config.authorized(response, request, request.URL.Query().Get("institutionId")) {
		return
	}
	if config.auditEventRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "audit event repository is not configured")
		return
	}

	filter, err := parseListAuditEventsFilter(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	page, err := config.auditEventRepository.ListAuditEvents(request.Context(), filter)
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listAuditEventsResponse{
		Events:     toAuditEventResponses(page.Events),
		Pagination: toPaginationResponse(filter.Pagination, page.Total),
	})
}

func (config routerConfig) listInstitutions(response http.ResponseWriter, request *http.Request) {
	if !config.authorized(response, request, "") {
		return
	}
	if config.institutionRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "institution repository is not configured")
		return
	}

	filter, err := parseListInstitutionsFilter(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	page, err := config.institutionRepository.ListInstitutions(request.Context(), filter)
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listInstitutionsResponse{
		Institutions: toInstitutionResponses(page.Institutions),
		Pagination:   toPaginationResponse(filter.Pagination, page.Total),
	})
}

type paginationResponse struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

func toPaginationResponse(pagination treasury.Pagination, total int) paginationResponse {
	return paginationResponse{Page: pagination.Page, PageSize: pagination.PageSize, Total: total}
}

type listTransactionsResponse struct {
	Transactions []transactionResponse `json:"transactions"`
	Pagination   paginationResponse    `json:"pagination"`
}

type listAuditEventsResponse struct {
	Events     []auditEventResponse `json:"events"`
	Pagination paginationResponse   `json:"pagination"`
}

type listInstitutionsResponse struct {
	Institutions []institutionResponse `json:"institutions"`
	Pagination   paginationResponse    `json:"pagination"`
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

const (
	maxListLimit    = 100
	defaultPageSize = 15
)

var errInvalidPositiveInt = errors.New("invalid value")

func parsePositiveInt(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, errInvalidPositiveInt
	}
	return value, nil
}

// parseLimit enforces the documented 1..100 range strictly; out-of-range
// values are rejected rather than silently clamped (ADR-0002).
func parseLimit(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 || value > maxListLimit {
		return 0, treasury.ErrInvalidLimit
	}

	return value, nil
}

// parsePagination reads page/pageSize (with `limit` as a deprecated alias
// for pageSize). Table state travels in query parameters by convention.
func parsePagination(query url.Values) (treasury.Pagination, error) {
	pagination := treasury.Pagination{Page: 1, PageSize: defaultPageSize}

	if raw := query.Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return treasury.Pagination{}, treasury.ErrInvalidPage
		}
		pagination.Page = value
	}

	if raw := query.Get("pageSize"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 || value > maxListLimit {
			return treasury.Pagination{}, treasury.ErrInvalidPageSize
		}
		pagination.PageSize = value
	} else if raw := query.Get("limit"); raw != "" {
		value, err := parseLimit(raw)
		if err != nil {
			return treasury.Pagination{}, err
		}
		pagination.PageSize = value
	}

	return pagination, nil
}

// parseDateRange reads inclusive ISO-date bounds named fromKey/toKey and
// rejects malformed or inverted ranges.
func parseDateRange(query url.Values) (from, to string, err error) {
	from = query.Get("dateFrom")
	to = query.Get("dateTo")

	for _, value := range []string{from, to} {
		if value == "" {
			continue
		}
		if _, parseErr := time.Parse(time.DateOnly, value); parseErr != nil {
			return "", "", treasury.ErrInvalidDateRange
		}
	}

	if from != "" && to != "" && from > to {
		return "", "", treasury.ErrInvalidDateRange
	}

	return from, to, nil
}

// parseAmountBound reads a non-negative minor-unit integer filter value.
func parseAmountBound(query url.Values, key string) (int64, error) {
	raw := query.Get(key)
	if raw == "" {
		return 0, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, treasury.ErrInvalidAmountFilter
	}

	return value, nil
}

func parseListTransactionsFilter(request *http.Request) (treasury.ListTransactionsFilter, error) {
	query := request.URL.Query()
	filter := treasury.ListTransactionsFilter{
		InstitutionID: query.Get("institutionId"),
	}

	if fiscalYear := query.Get("fiscalYear"); fiscalYear != "" {
		value, err := strconv.Atoi(fiscalYear)
		if err != nil || value <= 0 {
			return treasury.ListTransactionsFilter{}, treasury.ErrInvalidFiscalYear
		}
		filter.FiscalYear = value
	}

	var err error
	if filter.DateFrom, filter.DateTo, err = parseDateRange(query); err != nil {
		return treasury.ListTransactionsFilter{}, err
	}

	if filter.AmountMinorGte, err = parseAmountBound(query, "amountGte"); err != nil {
		return treasury.ListTransactionsFilter{}, err
	}

	if filter.AmountMinorLte, err = parseAmountBound(query, "amountLte"); err != nil {
		return treasury.ListTransactionsFilter{}, err
	}

	if filter.Pagination, err = parsePagination(query); err != nil {
		return treasury.ListTransactionsFilter{}, err
	}

	return filter, nil
}

func parseListAuditEventsFilter(request *http.Request) (treasury.ListAuditEventsFilter, error) {
	query := request.URL.Query()
	filter := treasury.ListAuditEventsFilter{
		InstitutionID: query.Get("institutionId"),
	}

	var err error
	if filter.DateFrom, filter.DateTo, err = parseDateRange(query); err != nil {
		return treasury.ListAuditEventsFilter{}, err
	}

	if filter.Pagination, err = parsePagination(query); err != nil {
		return treasury.ListAuditEventsFilter{}, err
	}

	return filter, nil
}

var validInstitutionStatuses = map[string]struct{}{
	"ACTIVE":   {},
	"INACTIVE": {},
}

func parseListInstitutionsFilter(request *http.Request) (treasury.ListInstitutionsFilter, error) {
	query := request.URL.Query()
	filter := treasury.ListInstitutionsFilter{}

	if status := query.Get("status"); status != "" {
		if _, ok := validInstitutionStatuses[status]; !ok {
			return treasury.ListInstitutionsFilter{}, treasury.ErrInvalidStatus
		}
		filter.Status = status
	}

	var err error
	if filter.Pagination, err = parsePagination(query); err != nil {
		return treasury.ListInstitutionsFilter{}, err
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
