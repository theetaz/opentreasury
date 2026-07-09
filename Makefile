.PHONY: verify verify-go verify-integration verify-js verify-compose up down db-up db-down db-migrate db-migrate-down db-seed db-reset

PNPM ?= corepack pnpm
COMPOSE ?= docker compose -f infra/docker/docker-compose.yaml
MIGRATE_IMAGE ?= migrate/migrate:v4.19.1
DB_DSN ?= postgres://opentreasury:opentreasury@localhost:5433/opentreasury_dev?sslmode=disable
# host.docker.internal works on Docker Desktop (macOS/Windows); on Linux use --network host and localhost.
DB_DSN_FROM_CONTAINER ?= postgres://opentreasury:opentreasury@host.docker.internal:5433/opentreasury_dev?sslmode=disable

verify-compose:
	$(COMPOSE) config -q

# Full local stack: Postgres + migrations + seeds + core API + web app.
# Dashboard: http://localhost:5173  ·  API: http://localhost:8080
up:
	$(COMPOSE) up -d --build --wait postgres core-api web

down:
	$(COMPOSE) down

db-up:
	$(COMPOSE) up -d --wait postgres

db-down:
	$(COMPOSE) down

db-migrate: db-up
	docker run --rm -v $(PWD)/database/migrations:/migrations $(MIGRATE_IMAGE) \
		-path=/migrations -database "$(DB_DSN_FROM_CONTAINER)" up

db-migrate-down: db-up
	docker run --rm -v $(PWD)/database/migrations:/migrations $(MIGRATE_IMAGE) \
		-path=/migrations -database "$(DB_DSN_FROM_CONTAINER)" down -all

db-seed: db-migrate
	@for seed in database/seeds/*.sql; do \
		echo "==> applying $$seed"; \
		docker exec -i opentreasury-postgres psql -q -U opentreasury -d opentreasury_dev < $$seed; \
	done

db-reset:
	$(COMPOSE) down -v
	$(MAKE) db-seed

verify: verify-go verify-integration verify-js

verify-go:
	@for mod in services/core-api services/connectors/generic-file services/ledger-gateway services/mcp-server workers/treasury-worker chaincode/treasury; do \
		echo "==> go test ./... in $$mod"; \
		(cd $$mod && go test ./...); \
	done

# Ryuk (the testcontainers reaper) hangs on some Docker Desktop setups; tests
# terminate their containers explicitly in t.Cleanup, so the reaper is optional.
verify-integration:
	@echo "==> core API integration tests"
	@(cd services/core-api && TESTCONTAINERS_RYUK_DISABLED=true go test -tags=integration -timeout 10m ./test/integration)

verify-js:
	@if [ -d node_modules ]; then \
		$(PNPM) --filter @opentreasury/web test; \
		$(PNPM) --filter @opentreasury/web --filter @opentreasury/types --filter @opentreasury/ui typecheck; \
	else \
		echo "Skipping JS verification until pnpm dependencies are installed."; \
	fi
