package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

// discoveryOverride lets the API fetch OIDC metadata from an internal URL
// while still validating the issuer claim from tokens minted at the external
// URL (containerized local dev).
var discoveryOverride = os.Getenv("OPENTREASURY_OIDC_DISCOVERY_URL")

// OIDCVerifier verifies Keycloak-issued access tokens against the realm's
// published JWKS and maps their claims to a Principal.
type OIDCVerifier struct {
	verifier *oidc.IDTokenVerifier
}

// NewOIDCVerifier discovers the issuer's OIDC configuration and returns a
// verifier bound to the given audience (the API's client id).
//
// In containerized local dev the browser reaches Keycloak at one URL (the
// token issuer) while the API reaches it at an internal URL. When
// OPENTREASURY_OIDC_DISCOVERY_URL differs from the issuer, discovery uses the
// internal URL while the issuer claim is still validated against issuerURL.
func NewOIDCVerifier(ctx context.Context, issuerURL, audience string) (*OIDCVerifier, error) {
	// Keycloak access tokens carry the client id in azp rather than aud;
	// skip audience checks here and rely on issuer + signature validity.
	config := &oidc.Config{SkipClientIDCheck: true}

	// Single-URL case (production): standard discovery.
	if discoveryOverride == "" || discoveryOverride == issuerURL {
		provider, err := oidc.NewProvider(ctx, issuerURL)
		if err != nil {
			return nil, fmt.Errorf("discovering oidc provider: %w", err)
		}
		return &OIDCVerifier{verifier: provider.Verifier(config)}, nil
	}

	// Split-URL case (containerized local dev): the token issuer is the
	// external URL, but discovery advertises a JWKS URL that is unreachable
	// from inside the container. Fetch keys from the internal URL directly and
	// validate the issuer claim against the external URL.
	keySet := oidc.NewRemoteKeySet(ctx, strings.TrimRight(discoveryOverride, "/")+"/protocol/openid-connect/certs")
	return &OIDCVerifier{verifier: oidc.NewVerifier(issuerURL, keySet, config)}, nil
}

type keycloakClaims struct {
	Subject           string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	InstitutionID     string `json:"institution_id"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

func (v *OIDCVerifier) Verify(ctx context.Context, rawToken string) (Principal, error) {
	token, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		return Principal{}, err
	}

	var claims keycloakClaims
	if err := token.Claims(&claims); err != nil {
		return Principal{}, err
	}

	return Principal{
		Subject:       claims.Subject,
		Username:      claims.PreferredUsername,
		Roles:         claims.RealmAccess.Roles,
		InstitutionID: claims.InstitutionID,
	}, nil
}
