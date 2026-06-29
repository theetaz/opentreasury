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

func TestListInstitutionsEndpointReturnsInstitutions(t *testing.T) {
	repository := &recordingInstitutionRepository{
		institutions: []treasury.Institution{
			{
				ID:          "minfin",
				Name:        "Ministry of Finance",
				Type:        "MINISTRY",
				CountryCode: "KE",
				Status:      "ACTIVE",
			},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/institutions?limit=25", nil)
	response := httptest.NewRecorder()

	NewRouter(WithInstitutionRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, treasury.ListInstitutionsFilter{Limit: 25}, repository.filter)

	var body struct {
		Institutions []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Type        string `json:"type"`
			CountryCode string `json:"countryCode"`
			Status      string `json:"status"`
		} `json:"institutions"`
	}
	err := json.NewDecoder(response.Body).Decode(&body)
	require.NoError(t, err)
	require.Len(t, body.Institutions, 1)
	require.Equal(t, "minfin", body.Institutions[0].ID)
	require.Equal(t, "Ministry of Finance", body.Institutions[0].Name)
	require.Equal(t, "MINISTRY", body.Institutions[0].Type)
	require.Equal(t, "KE", body.Institutions[0].CountryCode)
	require.Equal(t, "ACTIVE", body.Institutions[0].Status)
}

type recordingInstitutionRepository struct {
	institutions []treasury.Institution
	filter       treasury.ListInstitutionsFilter
}

func (repository *recordingInstitutionRepository) ListInstitutions(ctx context.Context, filter treasury.ListInstitutionsFilter) ([]treasury.Institution, error) {
	repository.filter = filter
	return repository.institutions, nil
}
