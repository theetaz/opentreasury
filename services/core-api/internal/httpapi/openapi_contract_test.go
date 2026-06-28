package httpapi

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIContractDocumentsCoreEndpoints(t *testing.T) {
	contractPath := filepath.Join("..", "..", "..", "..", "api", "core-api", "openapi.yaml")
	loader := openapi3.NewLoader()

	document, err := loader.LoadFromFile(contractPath)

	require.NoError(t, err)
	require.NoError(t, document.Validate(context.Background()))
	require.NotNil(t, document.Paths.Find("/healthz").Get)
	require.NotNil(t, document.Paths.Find("/v1/transactions/validate").Post)
	require.NotNil(t, document.Paths.Find("/v1/transactions").Post)
}
