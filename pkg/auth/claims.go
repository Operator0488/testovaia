package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims — JWT claims, которые выдаёт identity-service.
type Claims struct {
	Subject   string // UUID Identity
	Scope     Scope  // internal / external
	Issuer    string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// jwtClaims — внутренняя структура для парсинга JWT.
type jwtClaims struct {
	Scope string `json:"scope"`
	jwt.RegisteredClaims
}

// ParseToken проверяет подпись JWT (RS256) через KeyStore и возвращает claims.
// Проверяет: kid (key id) в header, подпись (только RSA PKCS#1) и срок действия (exp).
func ParseToken(keys KeyStore, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("неожиданный алгоритм подписи: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("kid отсутствует в header токена")
		}

		return keys.GetKey(kid)
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	raw, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, errors.New("некорректные claims токена")
	}

	scope := Scope(raw.Scope)
	if !scope.Valid() {
		return nil, fmt.Errorf("недопустимый scope: %q", raw.Scope)
	}

	c := &Claims{
		Subject: raw.Subject,
		Scope:   scope,
		Issuer:  raw.Issuer,
	}
	if raw.IssuedAt != nil {
		c.IssuedAt = raw.IssuedAt.Time
	}
	if raw.ExpiresAt != nil {
		c.ExpiresAt = raw.ExpiresAt.Time
	}

	return c, nil
}
