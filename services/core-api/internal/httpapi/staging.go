package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type StagingRepository interface {
	Ingest(context.Context, treasury.StagingRecord) (treasury.StagingOutcome, error)
	ListStagingRecords(context.Context, treasury.ListStagingRecordsFilter) (treasury.StagingRecordPage, error)
}

func WithStagingRepository(repository StagingRepository) RouterOption {
	return func(config *routerConfig) {
		config.stagingRepository = repository
	}
}

type stagingLinePayload struct {
	AccountCode string `json:"accountCode"`
	Direction   string `json:"direction"`
	AmountMinor int64  `json:"amountMinor"`
	Currency    string `json:"currency"`
}

type stagingRecordPayload struct {
	SourceSystem  string               `json:"sourceSystem"`
	SourceRef     string               `json:"sourceRef"`
	SourceHash    string               `json:"sourceHash"`
	Profile       string               `json:"profile"`
	InstitutionID string               `json:"institutionId"`
	OccurredAt    string               `json:"occurredAt"`
	Lines         []stagingLinePayload `json:"lines"`
}

type stagingBatchPayload struct {
	Records []stagingRecordPayload `json:"records"`
}

type stagingOutcomeResponse struct {
	SourceRef  string `json:"sourceRef"`
	SourceHash string `json:"sourceHash"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	EntryID    string `json:"entryId,omitempty"`
}

const maxStagingBatch = 500

func (config routerConfig) submitStagingRecords(response http.ResponseWriter, request *http.Request) {
	if config.stagingRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "staging repository is not configured")
		return
	}

	var payload stagingBatchPayload
	if !decodeRequestBody(response, request, &payload) {
		return
	}

	if len(payload.Records) == 0 {
		writeError(response, http.StatusBadRequest, "batch contains no records")
		return
	}
	if len(payload.Records) > maxStagingBatch {
		writeError(response, http.StatusBadRequest, "batch exceeds 500 records")
		return
	}

	// Authorize once per distinct institution in the batch.
	seen := map[string]struct{}{}
	for _, record := range payload.Records {
		if _, ok := seen[record.InstitutionID]; ok {
			continue
		}
		seen[record.InstitutionID] = struct{}{}
		if !config.authorized(response, request, record.InstitutionID) {
			return
		}
	}

	ctx := treasury.ContextWithAuditMetadata(request.Context(), treasury.AuditMetadata{
		Actor:     "connector",
		RequestID: requestIDFromContext(request.Context()),
	})

	outcomes := make([]stagingOutcomeResponse, 0, len(payload.Records))
	for _, record := range payload.Records {
		if record.SourceHash == "" || record.SourceSystem == "" || record.SourceRef == "" {
			outcomes = append(outcomes, stagingOutcomeResponse{
				SourceRef:  record.SourceRef,
				SourceHash: record.SourceHash,
				Status:     "QUARANTINED",
				Reason:     "missing source identity (sourceSystem, sourceRef, sourceHash)",
			})
			continue
		}

		lines := make([]treasury.JournalLine, 0, len(record.Lines))
		for _, line := range record.Lines {
			lines = append(lines, treasury.JournalLine{
				AccountCode: line.AccountCode,
				Direction:   line.Direction,
				AmountMinor: line.AmountMinor,
				Currency:    line.Currency,
			})
		}

		outcome, err := config.stagingRepository.Ingest(ctx, treasury.StagingRecord{
			SourceSystem:  record.SourceSystem,
			SourceRef:     record.SourceRef,
			SourceHash:    record.SourceHash,
			Profile:       record.Profile,
			InstitutionID: record.InstitutionID,
			OccurredAt:    record.OccurredAt,
			Lines:         lines,
		})
		if err != nil {
			config.internalError(response, request, err)
			return
		}

		outcomes = append(outcomes, stagingOutcomeResponse{
			SourceRef:  outcome.SourceRef,
			SourceHash: outcome.SourceHash,
			Status:     outcome.Status,
			Reason:     outcome.Reason,
			EntryID:    outcome.EntryID,
		})
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(map[string]any{"outcomes": outcomes})
}

type stagingRecordResponse struct {
	ID            string                `json:"id"`
	SourceSystem  string                `json:"sourceSystem"`
	SourceRef     string                `json:"sourceRef"`
	Profile       string                `json:"profile"`
	InstitutionID string                `json:"institutionId"`
	OccurredAt    string                `json:"occurredAt"`
	Status        string                `json:"status"`
	Reason        string                `json:"reason,omitempty"`
	EntryID       string                `json:"entryId,omitempty"`
	CreatedAt     string                `json:"createdAt"`
	Lines         []journalLineResponse `json:"lines"`
}

var validStagingStatuses = map[string]struct{}{
	"POSTED":      {},
	"QUARANTINED": {},
}

func (config routerConfig) listStagingRecords(response http.ResponseWriter, request *http.Request) {
	if !config.authorized(response, request, request.URL.Query().Get("institutionId")) {
		return
	}
	if config.stagingRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "staging repository is not configured")
		return
	}

	query := request.URL.Query()
	filter := treasury.ListStagingRecordsFilter{
		InstitutionID: query.Get("institutionId"),
		SourceSystem:  query.Get("sourceSystem"),
	}

	if status := query.Get("status"); status != "" {
		if _, ok := validStagingStatuses[status]; !ok {
			writeError(response, http.StatusBadRequest, treasury.ErrInvalidStatus.Error())
			return
		}
		filter.Status = status
	}

	pagination, err := parsePagination(query)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}
	filter.Pagination = pagination

	page, err := config.stagingRepository.ListStagingRecords(request.Context(), filter)
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	records := make([]stagingRecordResponse, 0, len(page.Records))
	for _, record := range page.Records {
		lines := make([]journalLineResponse, 0, len(record.Lines))
		for _, line := range record.Lines {
			lines = append(lines, journalLineResponse{
				AccountCode: line.AccountCode,
				Direction:   line.Direction,
				AmountMinor: line.AmountMinor,
				Currency:    line.Currency,
			})
		}
		records = append(records, stagingRecordResponse{
			ID:            record.ID,
			SourceSystem:  record.SourceSystem,
			SourceRef:     record.SourceRef,
			Profile:       record.Profile,
			InstitutionID: record.InstitutionID,
			OccurredAt:    record.OccurredAt,
			Status:        record.Status,
			Reason:        record.Reason,
			EntryID:       record.EntryID,
			CreatedAt:     record.CreatedAt,
			Lines:         lines,
		})
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(map[string]any{
		"records":    records,
		"pagination": toPaginationResponse(filter.Pagination, page.Total),
	})
}
