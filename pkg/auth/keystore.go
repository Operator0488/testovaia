package auth

import (
	"crypto/rsa"
	"fmt"
)

// KeyStore хранит публичные ключи для верификации JWT, индексированные по kid.
type KeyStore interface {
	GetKey(kid string) (*rsa.PublicKey, error)
}

// StaticKeyStore — неизменяемое хранилище ключей в памяти (в последующем заменим на кеш над JWKS.json)
type StaticKeyStore struct {
	keys map[string]*rsa.PublicKey
}

// NewStaticKeyStore создаёт хранилище из словаря kid → публичный ключ.
func NewStaticKeyStore(keys map[string]*rsa.PublicKey) *StaticKeyStore {
	return &StaticKeyStore{keys: keys}
}

func (s *StaticKeyStore) GetKey(kid string) (*rsa.PublicKey, error) {
	key, ok := s.keys[kid]
	if !ok {
		return nil, fmt.Errorf("неизвестный kid: %q", kid)
	}
	return key, nil
}
