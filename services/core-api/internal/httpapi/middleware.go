package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

type contextKey string

const requestIDContextKey contextKey = "requestID"

const requestIDHeader = "X-Request-ID"

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get(requestIDHeader)
		if !isValidRequestID(requestID) {
			requestID = newRequestID()
		}

		response.Header().Set(requestIDHeader, requestID)
		ctx := context.WithValue(request.Context(), requestIDContextKey, requestID)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

func isValidRequestID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}

	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
		case char >= 'A' && char <= 'Z':
		case char >= '0' && char <= '9':
		case char == '-' || char == '_' || char == '.':
		default:
			return false
		}
	}

	return true
}

func newRequestID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		// crypto/rand failure means the process is in a hopeless state;
		// a constant keeps requests flowing rather than panicking.
		return "unavailable"
	}

	return hex.EncodeToString(buffer)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func withRequestLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: response, status: http.StatusOK}

		next.ServeHTTP(recorder, request)

		logger.Info("request",
			"request_id", requestIDFromContext(request.Context()),
			"method", request.Method,
			"path", request.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func withRecovery(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered",
					"request_id", requestIDFromContext(request.Context()),
					"method", request.Method,
					"path", request.URL.Path,
					"panic", recovered,
					"stack", string(debug.Stack()),
				)
				writeError(response, http.StatusInternalServerError, "internal server error")
			}
		}()

		next.ServeHTTP(response, request)
	})
}

func withMaxBodyBytes(limit int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Body != nil && request.ContentLength != 0 {
			request.Body = http.MaxBytesReader(response, request.Body, limit)
		}

		next.ServeHTTP(response, request)
	})
}
