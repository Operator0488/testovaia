package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/auth"
)

func TestClaimsContext(t *testing.T) {
	t.Run("достаёт claims из контекста", func(t *testing.T) {
		claims := &auth.Claims{
			Subject:   "user-uuid-123",
			Scope:     auth.ScopeExternal,
			Issuer:    "identity-service",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}

		ctx := auth.WithClaims(context.Background(), claims)
		got := auth.ClaimsFromContext(ctx)

		assert.Equal(t, claims, got)
	})

	t.Run("возвращает nil если claims не были положены", func(t *testing.T) {
		got := auth.ClaimsFromContext(context.Background())
		assert.Nil(t, got)
	})

	t.Run("дочерний контекст наследует claims", func(t *testing.T) {
		claims := &auth.Claims{Subject: "abc"}
		ctx := auth.WithClaims(context.Background(), claims)
		child := context.WithValue(ctx, "unrelated", "value") //nolint:staticcheck

		got := auth.ClaimsFromContext(child)
		assert.Equal(t, claims, got)
	})
}
