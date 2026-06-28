.PHONY: verify verify-go verify-js

verify: verify-go verify-js

verify-go:
	@for mod in services/core-api services/connectors/generic-file services/ledger-gateway services/mcp-server workers/treasury-worker chaincode/treasury; do \
		echo "==> go test ./... in $$mod"; \
		(cd $$mod && go test ./...); \
	done

verify-js:
	@if command -v pnpm >/dev/null 2>&1 && [ -d node_modules ]; then \
		pnpm -r typecheck; \
	else \
		echo "Skipping JS verification until pnpm dependencies are installed."; \
	fi
