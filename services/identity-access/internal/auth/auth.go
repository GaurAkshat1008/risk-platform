package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Validator struct {
	verifier *oidc.IDTokenVerifier
	expectedIssuer string
	expectedAudience string
}

type Principal struct {
	UserID string
	TenantID string
	Roles []string
	ExpiresAt time.Time
	Issuer string
	Audience []string
}

type tokenClaims struct {
	Aud any `json:"aud"`
	Exp int64 `json:"exp"`
	Iss string `json:"iss"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	Sub string `json:"sub"`
	TenantID string `json:"tenant_id"`
}

func NewValidator(ctx context.Context, issuer, audience string) (*Validator, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})

	return &Validator{
		verifier: verifier,
		expectedIssuer: issuer,
		expectedAudience: audience,
	}, nil
}

func (v *Validator) ValidateAccessToken(ctx context.Context, token string) (Principal, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Principal{}, errors.New("access token is required")
	}

	idToken, err := v.verifier.Verify(ctx, token)
	if err != nil {
		return Principal{}, fmt.Errorf("failed to verify token: %w", err)
	}

	var c tokenClaims

	if err := idToken.Claims(&c); err != nil {
		return Principal{}, fmt.Errorf("failed to parse token claims: %w", err)
	}

	if c.Iss != v.expectedIssuer {
		return Principal{}, fmt.Errorf("invalid token issuer: expected %s, got %s", v.expectedIssuer, c.Iss)
	}

	if c.TenantID == "" {
		return Principal{}, errors.New("tenant_id claim is required")
	}

	aud := normalizeAudience(c.Aud)
	if v.expectedAudience != "" && !contains(aud, v.expectedAudience) {
		return Principal{}, fmt.Errorf("invalid token audience: expected %s, got %v", v.expectedAudience, aud)
	}
	return Principal{
		UserID: c.Sub,
		TenantID: c.TenantID,
		Roles: c.RealmAccess.Roles,
		ExpiresAt: time.Unix(c.Exp, 0),
		Issuer: c.Iss,
		Audience: aud,
	}, nil
}

func normalizeAudience(aud any) []string {
	switch t := aud.(type) {
	case string: 
		return []string{t}
	case []interface{}:
		var result []string
		for _, v := range t {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func contains(aud []string, expected string) bool {
	for _, a := range aud {
		if a == expected {
			return true
		}
	}
	return false
}