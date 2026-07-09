// The OpenTreasury MCP server exposes public treasury data as governed tools
// for AI agents, over the streamable HTTP transport. Every tool reads from the
// core API's anonymous public tier, so the publication policy bounds what any
// agent can see.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/opentreasury/opentreasury/services/mcp-server/internal/tools"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	addr := os.Getenv("OPENTREASURY_MCP_ADDR")
	if addr == "" {
		addr = ":8090"
	}
	apiBaseURL := os.Getenv("OPENTREASURY_PUBLIC_API_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8080"
	}

	client := tools.NewClient(apiBaseURL)

	newServer := func(*http.Request) *mcp.Server {
		server := mcp.NewServer(&mcp.Implementation{
			Name:    "opentreasury",
			Version: "0.1.0",
		}, nil)
		tools.Register(server, client)
		return server
	}

	handler := mcp.NewStreamableHTTPHandler(newServer, nil)

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("mcp server listening", "addr", addr, "public_api", apiBaseURL)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("mcp server exited", "error", err)
		os.Exit(1)
	}
}
