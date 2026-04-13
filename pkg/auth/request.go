package auth

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
)

var (
	// ErrNotAuthenticated — claims отсутствуют в контексте запроса (пользователь не аутентифицирован).
	ErrNotAuthenticated = errors.New("пользователь не аутентифицирован")
	// ErrForbidden — пользователь аутентифицирован, но не соответствует требуемому scope.
	ErrForbidden = errors.New("недостаточно прав")
)

// IdentityFromRequest возвращает UUID пользователя из JWT-claims запроса.
// Если claims отсутствуют или subject не является валидным UUID — возвращает ErrNotAuthenticated.
func IdentityFromRequest(r *http.Request) (uuid.UUID, error) {
	c := ClaimsFromContext(r.Context())
	if c == nil {
		return uuid.Nil, ErrNotAuthenticated
	}

	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, ErrNotAuthenticated
	}

	return id, nil
}

// IsInScope проверяет, что аутентифицированный пользователь обладает указанным scope.
// Возвращает ErrNotAuthenticated если claims отсутствуют, ErrForbidden если scope не совпадает.
func IsInScope(r *http.Request, scope Scope) error {
	c := ClaimsFromContext(r.Context())
	if c == nil {
		return ErrNotAuthenticated
	}

	if c.Scope != scope {
		return ErrForbidden
	}

	return nil
}
