package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// stubPublicAPI mimics the core API's public tier.
func stubPublicAPI(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /public/v1/institutions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"institutions":[{"id":"minfin","name":"Ministry of Finance"}],"pagination":{"page":1,"pageSize":15,"total":1}}`))
	})
	mux.HandleFunc("GET /public/v1/balances", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("institutionId") != "minfin" {
			t.Errorf("institutionId filter not forwarded, got %q", r.URL.Query().Get("institutionId"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"balances":[{"institutionId":"minfin","accountCode":"6202","balanceMinor":500000,"currency":"USD"}],"pagination":{"page":1,"pageSize":15,"total":1}}`))
	})
	mux.HandleFunc("GET /public/v1/journal-entries", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"entries":[{"id":"je-1","lines":[{"accountCode":"6202","direction":"DEBIT","amountMinor":500000,"currency":"USD"}]}],"pagination":{"page":1,"pageSize":15,"total":1}}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// connect wires an in-memory MCP client to a server with our tools registered.
func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()

	api := stubPublicAPI(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "opentreasury-test", Version: "0.0.1"}, nil)
	Register(server, NewClient(api.URL))

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connecting client: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	return session
}

func TestToolsAreListed(t *testing.T) {
	session := connect(t)

	result, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("listing tools: %v", err)
	}

	names := map[string]bool{}
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"list_institutions", "list_accounts", "query_balances", "query_flows"} {
		if !names[want] {
			t.Errorf("tool %q not registered", want)
		}
	}
}

func TestQueryBalancesForwardsFiltersAndReturnsData(t *testing.T) {
	session := connect(t)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "query_balances",
		Arguments: map[string]any{"institutionId": "minfin"},
	})
	if err != nil {
		t.Fatalf("calling tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var payload struct {
		Balances []struct {
			BalanceMinor int64 `json:"balanceMinor"`
		} `json:"balances"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("tool result is not the API JSON: %v", err)
	}
	if len(payload.Balances) != 1 || payload.Balances[0].BalanceMinor != 500000 {
		t.Errorf("unexpected balances payload: %s", text)
	}
}

func TestQueryFlowsReturnsEntries(t *testing.T) {
	session := connect(t)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "query_flows",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("calling tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !json.Valid([]byte(text)) || !strings.Contains(text, `"id":"je-1"`) {
		t.Errorf("unexpected flows payload: %s", text)
	}
}
