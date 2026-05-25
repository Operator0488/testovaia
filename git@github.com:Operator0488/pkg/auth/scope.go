package auth

// Scope определяет уровень доступа пользователя.
type Scope string

const (
	ScopeInternal Scope = "internal"
	ScopeExternal Scope = "external"
)

func (s Scope) Valid() bool {
	switch s {
	case ScopeInternal, ScopeExternal:
		return true
	default:
		return false
	}
}

func (s Scope) String() string {
	return string(s)
}
