package auth

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
)

const defaultRefreshInterval = 5 * time.Minute

// JWKSKeyStore — реализация KeyStore, загружающая публичные ключи из JWKS-эндпоинта.
// Периодически обновляет набор ключей в фоне.
type JWKSKeyStore struct {
	url    string
	client *http.Client

	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey

	stopCh chan struct{}
}

// NewJWKSKeyStore создаёт хранилище, делает первый fetch ключей и запускает фоновый тикер обновления.
func NewJWKSKeyStore(url string, refreshInterval time.Duration) (*JWKSKeyStore, error) {
	if refreshInterval <= 0 {
		refreshInterval = defaultRefreshInterval
	}

	s := &JWKSKeyStore{
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
		stopCh: make(chan struct{}),
	}

	if err := s.refresh(); err != nil {
		return nil, fmt.Errorf("первичная загрузка JWKS: %w", err)
	}

	go s.runRefreshLoop(refreshInterval)

	return s, nil
}

// GetKey возвращает публичный ключ по kid.
func (s *JWKSKeyStore) GetKey(kid string) (*rsa.PublicKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.keys[kid]
	if !ok {
		return nil, fmt.Errorf("неизвестный kid: %q", kid)
	}
	return key, nil
}

// Stop останавливает фоновый тикер обновления ключей.
func (s *JWKSKeyStore) Stop() error {
	close(s.stopCh)
	return nil
}

func (s *JWKSKeyStore) refresh() error {
	resp, err := s.client.Get(s.url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("GET %s: %w", s.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: статус %d", s.url, resp.StatusCode)
	}

	var jwks jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		rsaKey, ok := k.Key.(*rsa.PublicKey)
		if !ok {
			continue // пропускаем не-RSA ключи
		}
		keys[k.KeyID] = rsaKey
	}

	s.mu.Lock()
	s.keys = keys
	s.mu.Unlock()

	return nil
}

func (s *JWKSKeyStore) runRefreshLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = s.refresh() // ошибки логировать снаружи (через компоненту)
		case <-s.stopCh:
			return
		}
	}
}
