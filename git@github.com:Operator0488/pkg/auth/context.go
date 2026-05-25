package auth

import "context"

type contextKey int

const claimsKey contextKey = iota

// WithClaims кладёт Claims в контекст запроса.
func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

// ClaimsFromContext достаёт Claims из контекста. Возвращает nil если claims отсутствуют.
func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}
