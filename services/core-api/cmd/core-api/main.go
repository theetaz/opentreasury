package main

import (
	"log"
	"net/http"

	"github.com/opentreasury/opentreasury/services/core-api/internal/httpapi"
)

type config struct {
	addr string
}

func main() {
	cfg := config{addr: ":8080"}
	if err := newServer(cfg).ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func newServer(cfg config) *http.Server {
	return &http.Server{
		Addr:    cfg.addr,
		Handler: httpapi.NewRouter(),
	}
}
