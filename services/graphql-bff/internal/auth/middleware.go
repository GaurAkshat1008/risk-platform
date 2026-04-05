package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const tokenKey contextKey = "bearer_token"

// Middleware extracts the Bearer token from the Authorization header
// and stores it in the request context for downstream resolver use.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r.Header.Get("Authorization"))
		if token != "" {
			r = r.WithContext(WithToken(r.Context(), token))
		}
		next.ServeHTTP(w, r)
	})
}

// WithToken stores the bearer token in the context.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// TokenFromContext retrieves the bearer token from the context.
// Returns empty string if not present.
func TokenFromContext(ctx context.Context) string {
	if t, ok := ctx.Value(tokenKey).(string); ok {
		return t
	}
	return ""
}

func extractBearerToken(authHeader string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(authHeader, prefix) {
		return authHeader[len(prefix):]
	}
	return ""
}
