package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/opentreasury/opentreasury/services/core-api/internal/httpapi"
	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type config struct {
	addr           string
	databaseDSN    string
	allowedOrigins []string
}

func main() {
	cfg := loadConfig()
	db, closeDatabase, err := openDatabase(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := closeDatabase(); err != nil {
			log.Print(err)
		}
	}()

	if err := newServer(cfg, db).ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func loadConfig() config {
	addr := os.Getenv("OPENTREASURY_CORE_API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	return config{
		addr:           addr,
		databaseDSN:    os.Getenv("OPENTREASURY_DATABASE_DSN"),
		allowedOrigins: splitCSV(os.Getenv("OPENTREASURY_ALLOWED_ORIGINS")),
	}
}

func newServer(cfg config, db *sql.DB) *http.Server {
	options := []httpapi.RouterOption{
		httpapi.WithAllowedOrigins(cfg.allowedOrigins),
	}
	if db != nil {
		options = append(
			options,
			httpapi.WithTransactionRepository(treasury.NewPostgresTransactionRepository(db)),
			httpapi.WithInstitutionRepository(treasury.NewPostgresInstitutionRepository(db)),
		)
	}

	return &http.Server{
		Addr:    cfg.addr,
		Handler: httpapi.NewRouter(options...),
	}
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}

	return values
}

func openDatabase(cfg config) (*sql.DB, func() error, error) {
	if cfg.databaseDSN == "" {
		return nil, func() error { return nil }, nil
	}

	db, err := sql.Open("pgx", cfg.databaseDSN)
	if err != nil {
		return nil, nil, err
	}

	return db, db.Close, nil
}
