package authz

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/auth"
)

func newAuthorizer(t *testing.T) *Authorizer {
	t.Helper()
	authorizer, err := New(context.Background(), slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	return authorizer
}

func TestAuthorizationDecisionMatrix(t *testing.T) {
	authorizer := newAuthorizer(t)

	treasury := auth.Principal{Subject: "t", Roles: []string{"treasury-admin"}}
	auditor := auth.Principal{Subject: "a", Roles: []string{"auditor"}}
	minfin := auth.Principal{Subject: "i", Roles: []string{"institution-user"}, InstitutionID: "minfin"}
	connector := auth.Principal{Subject: "c", Roles: []string{"connector"}}

	cases := []struct {
		name      string
		principal auth.Principal
		action    string
		resource  string
		target    string
		want      bool
	}{
		{"treasury reads any institution", treasury, "read", "balances", "health", true},
		{"treasury writes any institution", treasury, "write", "journal", "transport", true},
		{"auditor reads cross-institution", auditor, "read", "audit-events", "health", true},
		{"auditor cannot write", auditor, "write", "journal", "minfin", false},
		{"institution user reads own institution", minfin, "read", "transactions", "minfin", true},
		{"institution user cannot read another institution", minfin, "read", "transactions", "health", false},
		{"institution user writes own institution", minfin, "write", "journal", "minfin", true},
		{"institution user cannot write another institution", minfin, "write", "journal", "transport", false},
		{"institution user reads shared accounts", minfin, "read", "accounts", "", true},
		{"institution user reads unscoped list", minfin, "read", "balances", "", true},
		{"anonymous is denied", auth.Principal{}, "read", "balances", "minfin", false},
		{"connector submits staging for any institution", connector, "write", "staging", "health", true},
		{"connector reads staging", connector, "read", "staging", "", true},
		{"connector cannot write the journal directly", connector, "write", "journal", "minfin", false},
		{"connector cannot read balances", connector, "read", "balances", "", false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := authorizer.Allow(context.Background(), "req-test", Input{
				Principal:     testCase.principal,
				Action:        testCase.action,
				Resource:      testCase.resource,
				InstitutionID: testCase.target,
			})
			require.Equal(t, testCase.want, got)
		})
	}
}
