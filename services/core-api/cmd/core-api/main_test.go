package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServerUsesConfiguredAddressAndRouter(t *testing.T) {
	server := newServer(config{addr: "127.0.0.1:9090"})

	if server.Addr != "127.0.0.1:9090" {
		t.Fatalf("server.Addr = %q, want %q", server.Addr, "127.0.0.1:9090")
	}

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	server.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusOK)
	}
}
