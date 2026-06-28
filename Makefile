.PHONY: verify verify-go verify-integration verify-js

verify: verify-go verify-integration verify-js

verify-go:
	@for mod in services/core-api services/connectors/generic-file services/ledger-gateway services/mcp-server workers/treasury-worker chaincode/treasury; do \
		echo "==> go test ./... in $$mod"; \
		(cd $$mod && go test ./...); \
	done

verify-integration:
	@echo "==> core API integration smoke test"
	@(cd services/core-api && go test -tags=integration ./test/integration)

verify-js:
	@if command -v pnpm >/dev/null 2>&1 && [ -d node_modules ]; then \
		pnpm --filter @opentreasury/web test; \
		pnpm --filter @opentreasury/web --filter @opentreasury/types --filter @opentreasury/ui typecheck; \
	else \
		echo "Skipping JS verification until pnpm dependencies are installed."; \
	fi
