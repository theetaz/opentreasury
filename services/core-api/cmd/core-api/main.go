package main

import (
	"log"
	"net/http"
	"os"

	"github.com/opentreasury/opentreasury/services/core-api/internal/httpapi"
)

type config struct {
	addr string
}

func main() {
	cfg := loadConfig()
	if err := newServer(cfg).ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func loadConfig() config {
	addr := os.Getenv("OPENTREASURY_CORE_API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	return config{addr: addr}
}

func newServer(cfg config) *http.Server {
	return &http.Server{
		Addr:    cfg.addr,
		Handler: httpapi.NewRouter(),
	}
}
