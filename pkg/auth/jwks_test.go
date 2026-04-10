package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/auth"
)

// jwksServer поднимает тестовый HTTP-сервер, отдающий JWKS с указанными ключами.
func jwksServer(t *testing.T, keys map[string]*rsa.PublicKey) *httptest.Server {
	t.Helper()

	jwks := jose.JSONWebKeySet{}
	for kid, pub := range keys {
		jwks.Keys = append(jwks.Keys, jose.JSONWebKey{
			Key:   pub,
			KeyID: kid,
			Use:   "sig",
		})
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
}

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func TestJWKSKeyStore_GetKey(t *testing.T) {
	t.Run("возвращает ключ по kid", func(t *testing.T) {
		privKey := generateRSAKey(t)
		srv := jwksServer(t, map[string]*rsa.PublicKey{"key-1": &privKey.PublicKey})
		defer srv.Close()

		store, err := auth.NewJWKSKeyStore(srv.URL, 0)
		require.NoError(t, err)
		defer store.Stop() //nolint:errcheck

		got, err := store.GetKey("key-1")
		require.NoError(t, err)
		assert.Equal(t, &privKey.PublicKey, got)
	})

	t.Run("возвращает ошибку для неизвестного kid", func(t *testing.T) {
		privKey := generateRSAKey(t)
		srv := jwksServer(t, map[string]*rsa.PublicKey{"key-1": &privKey.PublicKey})
		defer srv.Close()

		store, err := auth.NewJWKSKeyStore(srv.URL, 0)
		require.NoError(t, err)
		defer store.Stop() //nolint:errcheck

		_, err = store.GetKey("unknown-kid")
		assert.ErrorContains(t, err, "unknown-kid")
	})

	t.Run("ошибка если сервер недоступен при старте", func(t *testing.T) {
		_, err := auth.NewJWKSKeyStore("http://127.0.0.1:19999/jwks.json", 0)
		assert.Error(t, err)
	})

	t.Run("ошибка если сервер вернул не-200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		_, err := auth.NewJWKSKeyStore(srv.URL, 0)
		assert.Error(t, err)
	})
}

func TestJWKSKeyStore_Refresh(t *testing.T) {
	t.Run("подхватывает новые ключи после обновления", func(t *testing.T) {
		key1 := generateRSAKey(t)
		key2 := generateRSAKey(t)

		// Начинаем с key-1, потом переключаем на key-2
		current := map[string]*rsa.PublicKey{"key-1": &key1.PublicKey}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jwks := jose.JSONWebKeySet{}
			for kid, pub := range current {
				jwks.Keys = append(jwks.Keys, jose.JSONWebKey{Key: pub, KeyID: kid, Use: "sig"})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(jwks)
		}))
		defer srv.Close()

		store, err := auth.NewJWKSKeyStore(srv.URL, 50*time.Millisecond)
		require.NoError(t, err)
		defer store.Stop() //nolint:errcheck

		// key-1 доступен
		_, err = store.GetKey("key-1")
		require.NoError(t, err)

		// Меняем ключи на сервере
		current = map[string]*rsa.PublicKey{"key-2": &key2.PublicKey}

		// Ждём обновления
		assert.Eventually(t, func() bool {
			_, err := store.GetKey("key-2")
			return err == nil
		}, time.Second, 10*time.Millisecond)
	})
}
