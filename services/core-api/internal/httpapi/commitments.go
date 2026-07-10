package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type CommitmentRepository interface {
	CreateCommitment(context.Context, treasury.Commitment) error
	ListCommitments(context.Context, treasury.ListCommitmentsFilter) (treasury.CommitmentPage, error)
}

func WithCommitmentRepository(repository CommitmentRepository) RouterOption {
	return func(config *routerConfig) {
		config.commitmentRepository = repository
	}
}

type commitmentPayload struct {
	ID            string `json:"id"`
	InstitutionID string `json:"institutionId"`
	FiscalYear    int    `json:"fiscalYear"`
	AccountCode   string `json:"accountCode"`
	Description   string `json:"description"`
	AmountMinor   int64  `json:"amountMinor"`
	Currency      string `json:"currency"`
	CommittedDate string `json:"committedDate"`
}

func (config routerConfig) createCommitment(response http.ResponseWriter, request *http.Request) {
	if config.commitmentRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "commitment repository is not configured")
		return
	}

	var payload commitmentPayload
	if !decodeRequestBody(response, request, &payload) {
		return
	}

	commitment := treasury.Commitment{
		ID:            payload.ID,
		InstitutionID: payload.InstitutionID,
		FiscalYear:    payload.FiscalYear,
		AccountCode:   payload.AccountCode,
		Description:   payload.Description,
		AmountMinor:   payload.AmountMinor,
		Currency:      payload.Currency,
		CommittedDate: payload.CommittedDate,
	}
	if err := treasury.ValidateCommitment(commitment); err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	if !config.authorized(response, request, commitment.InstitutionID) {
		return
	}

	ctx := treasury.ContextWithAuditMetadata(request.Context(), treasury.AuditMetadata{
		Actor:     "system",
		RequestID: requestIDFromContext(request.Context()),
	})

	if err := config.commitmentRepository.CreateCommitment(ctx, commitment); err != nil {
		switch {
		case errors.Is(err, treasury.ErrDuplicateCommitment):
			writeError(response, http.StatusConflict, "commitment already exists")
		case errors.Is(err, treasury.ErrUnknownInstitution):
			writeError(response, http.StatusUnprocessableEntity, treasury.ErrUnknownInstitution.Error())
		default:
			config.internalError(response, request, err)
		}
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(response).Encode(map[string]string{"id": commitment.ID})
}

func (config routerConfig) listCommitments(response http.ResponseWriter, request *http.Request) {
	if !config.authorized(response, request, request.URL.Query().Get("institutionId")) {
		return
	}
	if config.commitmentRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "commitment repository is not configured")
		return
	}

	query := request.URL.Query()
	filter := treasury.ListCommitmentsFilter{
		InstitutionID: query.Get("institutionId"),
	}

	if status := query.Get("status"); status != "" {
		if _, ok := treasury.ValidCommitmentStatuses[status]; !ok {
			writeError(response, http.StatusBadRequest, treasury.ErrInvalidStatus.Error())
			return
		}
		filter.Status = status
	}

	if fiscalYear := query.Get("fiscalYear"); fiscalYear != "" {
		value, err := strconv.Atoi(fiscalYear)
		if err != nil || value <= 0 {
			writeError(response, http.StatusBadRequest, treasury.ErrInvalidFiscalYear.Error())
			return
		}
		filter.FiscalYear = value
	}

	pagination, err := parsePagination(query)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}
	filter.Pagination = pagination

	page, err := config.commitmentRepository.ListCommitments(request.Context(), filter)
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	commitments := make([]commitmentResponse, 0, len(page.Commitments))
	for _, commitment := range page.Commitments {
		commitments = append(commitments, toCommitmentResponse(commitment))
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listCommitmentsResponse{
		Commitments: commitments,
		Pagination:  toPaginationResponse(filter.Pagination, page.Total),
	})
}

type listCommitmentsResponse struct {
	Commitments []commitmentResponse `json:"commitments"`
	Pagination  paginationResponse   `json:"pagination"`
}

type commitmentResponse struct {
	ID                   string `json:"id"`
	InstitutionID        string `json:"institutionId"`
	FiscalYear           int    `json:"fiscalYear"`
	AccountCode          string `json:"accountCode"`
	Description          string `json:"description"`
	AmountMinor          int64  `json:"amountMinor"`
	Currency             string `json:"currency"`
	CommittedDate        string `json:"committedDate"`
	Status               string `json:"status"`
	SettledAmountMinor   int64  `json:"settledAmountMinor"`
	RemainingAmountMinor int64  `json:"remainingAmountMinor"`
}

func toCommitmentResponse(commitment treasury.Commitment) commitmentResponse {
	return commitmentResponse{
		ID:                   commitment.ID,
		InstitutionID:        commitment.InstitutionID,
		FiscalYear:           commitment.FiscalYear,
		AccountCode:          commitment.AccountCode,
		Description:          commitment.Description,
		AmountMinor:          commitment.AmountMinor,
		Currency:             commitment.Currency,
		CommittedDate:        commitment.CommittedDate,
		Status:               commitment.Status,
		SettledAmountMinor:   commitment.SettledAmountMinor,
		RemainingAmountMinor: commitment.AmountMinor - commitment.SettledAmountMinor,
	}
}
