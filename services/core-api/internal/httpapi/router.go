package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/transactions/validate", validateTransaction)
	return mux
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

	response.WriteHeader(http.StatusNoContent)
}

func writeError(response http.ResponseWriter, statusCode int, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)
	_ = json.NewEncoder(response).Encode(map[string]string{
		"error": message,
	})
}
