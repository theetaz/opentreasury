package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type TransactionRepository interface {
	Save(context.Context, treasury.Transaction) error
}

type RouterOption func(*routerConfig)

type routerConfig struct {
	transactionRepository TransactionRepository
}

func WithTransactionRepository(repository TransactionRepository) RouterOption {
	return func(config *routerConfig) {
		config.transactionRepository = repository
	}
}

func NewRouter(options ...RouterOption) http.Handler {
	config := routerConfig{}
	for _, option := range options {
		option(&config)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("POST /v1/transactions", config.createTransaction)
	mux.HandleFunc("POST /v1/transactions/validate", validateTransaction)
	return mux
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

	response.WriteHeader(http.StatusNoContent)
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

func writeError(response http.ResponseWriter, statusCode int, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)
	_ = json.NewEncoder(response).Encode(map[string]string{
		"error": message,
	})
}
