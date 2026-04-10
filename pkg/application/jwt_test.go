package application

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/auth"
)

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

func TestJWTAuthFunc(t *testing.T) {
	privKey := generateTestKey(t)
	keyStore := auth.NewStaticKeyStore(map[string]*rsa.PublicKey{"kid-1": &privKey.PublicKey})

	fn := jwtAuthFunc(keyStore)

	t.Run("валидный токен — claims кладутся в контекст", func(t *testing.T) {
		tokenStr := signToken(t, privKey, "kid-1", jwt.MapClaims{
			"sub":   "user-uuid-42",
			"scope": "external",
			"iss":   "identity-service",
			"iat":   time.Now().Unix(),
			"exp":   time.Now().Add(time.Hour).Unix(),
		})

		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)

		newReq, err := fn(req)
		require.NoError(t, err)

		claims := auth.ClaimsFromContext(newReq.Context())
		require.NotNil(t, claims)
		assert.Equal(t, "user-uuid-42", claims.Subject)
		assert.Equal(t, auth.ScopeExternal, claims.Scope)
	})

	t.Run("отсутствует Authorization header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		_, err := fn(req)
		assert.Error(t, err)
	})

	t.Run("неверный формат header (нет Bearer)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		_, err := fn(req)
		assert.Error(t, err)
	})

	t.Run("истёкший токен", func(t *testing.T) {
		tokenStr := signToken(t, privKey, "kid-1", jwt.MapClaims{
			"sub":   "user-uuid-42",
			"scope": "external",
			"iss":   "identity-service",
			"iat":   time.Now().Add(-2 * time.Hour).Unix(),
			"exp":   time.Now().Add(-time.Hour).Unix(),
		})

		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)

		_, err := fn(req)
		assert.Error(t, err)
	})

	t.Run("неизвестный kid", func(t *testing.T) {
		otherKey := generateTestKey(t)
		tokenStr := signToken(t, otherKey, "unknown-kid", jwt.MapClaims{
			"sub":   "user",
			"scope": "external",
			"exp":   time.Now().Add(time.Hour).Unix(),
		})

		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)

		_, err := fn(req)
		assert.Error(t, err)
	})
}

func TestExtractBearer(t *testing.T) {
	t.Run("корректный Bearer токен", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer my-token")
		got, err := extractBearer(req)
		require.NoError(t, err)
		assert.Equal(t, "my-token", got)
	})

	t.Run("Bearer с любым регистром", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "BEARER my-token")
		got, err := extractBearer(req)
		require.NoError(t, err)
		assert.Equal(t, "my-token", got)
	})

	t.Run("пустой header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		_, err := extractBearer(req)
		assert.Error(t, err)
	})
}
