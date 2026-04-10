package application

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/auth"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/di"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
)

var jwtComponent = NewComponent("jwt", initJWT, Noop)

// WithJWT подключает JWT-аутентификацию через JWKS-эндпоинт identity-service.
// После инициализации все входящие HTTP-запросы (кроме инфра-путей) проверяют Bearer-токен.
// Claims доступны в контексте запроса через auth.ClaimsFromContext.
func WithJWT() Option {
	return func(app *Application) error {
		app.components.add(component(jwtComponent))
		return nil
	}
}

func initJWT(ctx context.Context, app *Application) error {
	jwksURL := app.config.GetJWKSURL()
	logger.Info(ctx, "JWT initialize", logger.String("jwks_url", jwksURL))

	keyStore, err := auth.NewJWKSKeyStore(jwksURL, 0)
	if err != nil {
		return fmt.Errorf("jwt: не удалось загрузить JWKS: %w", err)
	}

	app.Closer.Add(keyStore.Stop)
	di.Register[auth.KeyStore](ctx, keyStore)

	defaultAuth.mu.Lock()
	defaultAuth.fn = jwtAuthFunc(keyStore)
	defaultAuth.mu.Unlock()

	return nil
}

// jwtAuthFunc возвращает authFunc, которая извлекает Bearer-токен, валидирует его
// и кладёт claims в контекст запроса.
func jwtAuthFunc(keys auth.KeyStore) authFunc {
	return func(r *http.Request) (*http.Request, error) {
		tokenString, err := extractBearer(r)
		if err != nil {
			return nil, err
		}

		claims, err := auth.ParseToken(keys, tokenString)
		if err != nil {
			return nil, fmt.Errorf("невалидный токен: %w", err)
		}

		return r.WithContext(auth.WithClaims(r.Context(), claims)), nil
	}
}

func extractBearer(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", fmt.Errorf("отсутствует Authorization header")
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", fmt.Errorf("неверный формат Authorization header")
	}

	return parts[1], nil
}
