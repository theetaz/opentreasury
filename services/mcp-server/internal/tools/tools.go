// Package tools implements the OpenTreasury MCP tools. Every tool is a
// governed façade over the anonymous public tier — an agent can never see
// more than the publication policy allows, by construction.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client calls the core API's public tier.
type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) get(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}

	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("core api unreachable: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("core api returned %d: %s", response.StatusCode, string(body))
	}

	return body, nil
}

// Register adds every OpenTreasury tool to the server.
func Register(server *mcp.Server, client *Client) {
	type listInstitutionsArgs struct {
		Page     int `json:"page,omitempty" jsonschema:"1-based page number"`
		PageSize int `json:"pageSize,omitempty" jsonschema:"rows per page (max 100)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_institutions",
		Description: "List government institutions reporting into the treasury (public data).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listInstitutionsArgs) (*mcp.CallToolResult, any, error) {
		return jsonResult(client.get(ctx, "/public/v1/institutions", pagination(args.Page, args.PageSize)))
	})

	type listAccountsArgs struct {
		AccountType string `json:"accountType,omitempty" jsonschema:"filter by classification: ASSET, LIABILITY, NET_WORTH, REVENUE, or EXPENSE"`
		Page        int    `json:"page,omitempty" jsonschema:"1-based page number"`
		PageSize    int    `json:"pageSize,omitempty" jsonschema:"rows per page (max 100)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_accounts",
		Description: "List the active chart of accounts (GFSM 2014 aligned classification of public finances).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listAccountsArgs) (*mcp.CallToolResult, any, error) {
		query := pagination(args.Page, args.PageSize)
		if args.AccountType != "" {
			query.Set("accountType", args.AccountType)
		}
		return jsonResult(client.get(ctx, "/public/v1/accounts", query))
	})

	type queryBalancesArgs struct {
		InstitutionID string `json:"institutionId,omitempty" jsonschema:"filter by institution id (e.g. minfin)"`
		AccountCode   string `json:"accountCode,omitempty" jsonschema:"filter by chart-of-accounts code"`
		Page          int    `json:"page,omitempty" jsonschema:"1-based page number"`
		PageSize      int    `json:"pageSize,omitempty" jsonschema:"rows per page (max 100)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_balances",
		Description: "Query materialized account balances (stocks): the net posted position per institution, account, and currency. Amounts are integers in minor units (cents).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args queryBalancesArgs) (*mcp.CallToolResult, any, error) {
		query := pagination(args.Page, args.PageSize)
		if args.InstitutionID != "" {
			query.Set("institutionId", args.InstitutionID)
		}
		if args.AccountCode != "" {
			query.Set("accountCode", args.AccountCode)
		}
		return jsonResult(client.get(ctx, "/public/v1/balances", query))
	})

	type queryFlowsArgs struct {
		InstitutionID string `json:"institutionId,omitempty" jsonschema:"filter by institution id"`
		DateFrom      string `json:"dateFrom,omitempty" jsonschema:"inclusive lower bound, ISO date (YYYY-MM-DD)"`
		DateTo        string `json:"dateTo,omitempty" jsonschema:"inclusive upper bound, ISO date (YYYY-MM-DD)"`
		Page          int    `json:"page,omitempty" jsonschema:"1-based page number"`
		PageSize      int    `json:"pageSize,omitempty" jsonschema:"rows per page (max 100)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_flows",
		Description: "Query posted double-entry journal entries (flows) with their debit/credit lines. Public projection: amounts, accounts, and dates are included; free-text descriptions are redacted by policy. Amounts are integers in minor units (cents).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args queryFlowsArgs) (*mcp.CallToolResult, any, error) {
		query := pagination(args.Page, args.PageSize)
		if args.InstitutionID != "" {
			query.Set("institutionId", args.InstitutionID)
		}
		if args.DateFrom != "" {
			query.Set("dateFrom", args.DateFrom)
		}
		if args.DateTo != "" {
			query.Set("dateTo", args.DateTo)
		}
		return jsonResult(client.get(ctx, "/public/v1/journal-entries", query))
	})
}

func pagination(page, pageSize int) url.Values {
	query := url.Values{}
	if page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		query.Set("pageSize", strconv.Itoa(pageSize))
	}
	return query
}

func jsonResult(body json.RawMessage, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}, nil, nil
}
