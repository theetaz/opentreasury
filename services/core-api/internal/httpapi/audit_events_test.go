package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
	"github.com/stretchr/testify/require"
)

func TestListAuditEventsEndpointReturnsEvents(t *testing.T) {
	repository := &recordingAuditEventRepository{
		events: []treasury.AuditEvent{
			{
				ID:            "audit-txn-2026-0001-created",
				EventType:     "TRANSACTION_CREATED",
				TransactionID: "txn-2026-0001",
				InstitutionID: "minfin",
				OccurredAt:    "2026-06-28T10:24:28Z",
				Summary:       "Transaction txn-2026-0001 was created.",
			},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/audit-events?institutionId=minfin&limit=25", nil)
	response := httptest.NewRecorder()

	NewRouter(WithAuditEventRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, treasury.ListAuditEventsFilter{
		InstitutionID: "minfin",
		Limit:         25,
	}, repository.filter)

	var body struct {
		Events []struct {
			ID            string `json:"id"`
			EventType     string `json:"eventType"`
			TransactionID string `json:"transactionId"`
			InstitutionID string `json:"institutionId"`
			OccurredAt    string `json:"occurredAt"`
			Summary       string `json:"summary"`
		} `json:"events"`
	}
	err := json.NewDecoder(response.Body).Decode(&body)
	require.NoError(t, err)
	require.Len(t, body.Events, 1)
	require.Equal(t, "audit-txn-2026-0001-created", body.Events[0].ID)
	require.Equal(t, "TRANSACTION_CREATED", body.Events[0].EventType)
	require.Equal(t, "txn-2026-0001", body.Events[0].TransactionID)
	require.Equal(t, "minfin", body.Events[0].InstitutionID)
	require.Equal(t, "2026-06-28T10:24:28Z", body.Events[0].OccurredAt)
}

type recordingAuditEventRepository struct {
	events []treasury.AuditEvent
	filter treasury.ListAuditEventsFilter
}

func (repository *recordingAuditEventRepository) ListAuditEvents(ctx context.Context, filter treasury.ListAuditEventsFilter) ([]treasury.AuditEvent, error) {
	repository.filter = filter
	return repository.events, nil
}
