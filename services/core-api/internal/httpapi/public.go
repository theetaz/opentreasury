package httpapi

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/opentreasury/opentreasury/services/core-api/internal/authz"
	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

// WithPublication enables the anonymous public read tier. Every row is passed
// through the publication policy's field mask before serialization.
func WithPublication(publication *authz.Publication) RouterOption {
	return func(config *routerConfig) {
		config.publication = publication
	}
}

const publicCacheControl = "public, max-age=30"

// WithPublicRateLimit tunes the per-IP rate limit on the public tier
// (defaults: 10 rps, burst 20).
func WithPublicRateLimit(rps float64, burst int) RouterOption {
	return func(config *routerConfig) {
		config.publicRateRPS = rps
		config.publicRateBurst = burst
	}
}

func (config routerConfig) registerPublicRoutes(mux *http.ServeMux) {
	if config.publication == nil {
		return
	}

	limited := newIPRateLimiter(config.publicRateRPS, config.publicRateBurst)

	handle := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, limited.wrap(handler))
	}

	handle("GET /public/v1/institutions", config.publicInstitutions)
	handle("GET /public/v1/accounts", config.publicAccounts)
	handle("GET /public/v1/balances", config.publicBalances)
	handle("GET /public/v1/journal-entries", config.publicJournalEntries)
}

func (config routerConfig) writePublicList(response http.ResponseWriter, resource, key string, rows []map[string]any, pagination paginationResponse) {
	masked := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		masked = append(masked, config.publication.Redact(resource, row))
	}

	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", publicCacheControl)
	_ = json.NewEncoder(response).Encode(map[string]any{
		key:          masked,
		"pagination": pagination,
	})
}

func (config routerConfig) publicInstitutions(response http.ResponseWriter, request *http.Request) {
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

	rows := make([]map[string]any, 0, len(page.Institutions))
	for _, institution := range page.Institutions {
		rows = append(rows, map[string]any{
			"id":          institution.ID,
			"name":        institution.Name,
			"type":        institution.Type,
			"countryCode": institution.CountryCode,
			"status":      institution.Status,
		})
	}

	config.writePublicList(response, "institutions", "institutions", rows, toPaginationResponse(filter.Pagination, page.Total))
}

func (config routerConfig) publicAccounts(response http.ResponseWriter, request *http.Request) {
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

	rows := make([]map[string]any, 0, len(page.Accounts))
	for _, account := range page.Accounts {
		rows = append(rows, map[string]any{
			"code":        account.Code,
			"name":        account.Name,
			"accountType": account.AccountType,
			"parentCode":  account.ParentCode,
			"gfsmCode":    account.GfsmCode,
			"cofogCode":   account.CofogCode,
			"active":      account.Active,
			"depth":       account.Depth,
		})
	}

	config.writePublicList(response, "accounts", "accounts", rows, toPaginationResponse(filter.Pagination, page.Total))
}

func (config routerConfig) publicBalances(response http.ResponseWriter, request *http.Request) {
	if config.journalRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "balance repository is not configured")
		return
	}

	query := request.URL.Query()
	pagination, err := parsePagination(query)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}
	filter := treasury.ListBalancesFilter{
		InstitutionID: query.Get("institutionId"),
		AccountCode:   query.Get("accountCode"),
		Pagination:    pagination,
	}

	page, err := config.journalRepository.ListBalances(request.Context(), filter)
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	rows := make([]map[string]any, 0, len(page.Balances))
	for _, balance := range page.Balances {
		rows = append(rows, map[string]any{
			"institutionId": balance.InstitutionID,
			"accountCode":   balance.AccountCode,
			"accountName":   balance.AccountName,
			"accountType":   balance.AccountType,
			"currency":      balance.Currency,
			"balanceMinor":  balance.BalanceMinor,
		})
	}

	config.writePublicList(response, "balances", "balances", rows, toPaginationResponse(filter.Pagination, page.Total))
}

func (config routerConfig) publicJournalEntries(response http.ResponseWriter, request *http.Request) {
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

	rows := make([]map[string]any, 0, len(page.Entries))
	for _, entry := range page.Entries {
		lines := make([]map[string]any, 0, len(entry.Lines))
		for _, line := range entry.Lines {
			lines = append(lines, map[string]any{
				"accountCode": line.AccountCode,
				"direction":   line.Direction,
				"amountMinor": line.AmountMinor,
				"currency":    line.Currency,
			})
		}
		rows = append(rows, map[string]any{
			"id":              entry.ID,
			"institutionId":   entry.InstitutionID,
			"fiscalYear":      entry.FiscalYear,
			"effectiveDate":   entry.EffectiveDate,
			"description":     entry.Description, // masked by policy
			"status":          entry.Status,
			"entryType":       entry.EntryType,
			"reversesEntryId": entry.ReversesEntryID,
			"idempotencyKey":  entry.IdempotencyKey, // masked by policy
			"lines":           lines,
		})
	}

	config.writePublicList(response, "journal-entries", "entries", rows, toPaginationResponse(filter.Pagination, page.Total))
}

// ipRateLimiter applies a per-client token bucket to the public tier so a
// single client cannot exhaust it. Stale buckets are swept periodically.
type ipRateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucketEntry
	rps      rate.Limit
	burst    int
	lastSwep time.Time
}

type bucketEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter(rps float64, burst int) *ipRateLimiter {
	if rps <= 0 {
		rps = 10
	}
	if burst <= 0 {
		burst = 20
	}
	return &ipRateLimiter{
		buckets:  map[string]*bucketEntry{},
		rps:      rate.Limit(rps),
		burst:    burst,
		lastSwep: time.Now(),
	}
}

func (l *ipRateLimiter) allow(clientIP string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.lastSwep) > 10*time.Minute {
		for ip, entry := range l.buckets {
			if now.Sub(entry.lastSeen) > 10*time.Minute {
				delete(l.buckets, ip)
			}
		}
		l.lastSwep = now
	}

	entry, ok := l.buckets[clientIP]
	if !ok {
		entry = &bucketEntry{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.buckets[clientIP] = entry
	}
	entry.lastSeen = now

	return entry.limiter.Allow()
}

func (l *ipRateLimiter) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		clientIP, _, err := net.SplitHostPort(request.RemoteAddr)
		if err != nil {
			clientIP = request.RemoteAddr
		}

		if !l.allow(clientIP) {
			response.Header().Set("Retry-After", "1")
			writeError(response, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(response, request)
	})
}
