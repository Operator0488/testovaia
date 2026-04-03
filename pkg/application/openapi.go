package application

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/swagger"
	"github.com/getkin/kin-openapi/openapi3"
)

const uuidPattern = `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`

var openapiComponent = NewComponent("openapi", initOpenAPI, Noop)
var registerUUIDOnce sync.Once

// RegisterFn — функция, которую сервис передаёт для регистрации своих HTTP-хендлеров на mux.
// ctx содержит DI-контейнер, что позволяет использовать di.Resolve для получения зависимостей.
type RegisterFn func(ctx context.Context, mux *http.ServeMux)

type specFile struct {
	Name string
	Data []byte
}

// WithOpenAPI — основная опция для REST API сервисов.
// Принимает fs.FS (обычно embed.FS), содержащий одну или несколько OpenAPI YAML-спецификаций.
// Автоматически запускает HTTP-сервер
//
// Настраивает:
//   - Middleware валидации запросов по всем спецификациям (kin-openapi); при ошибке — HTTP 400
//     и JSON {"error":{"trace_id","message"}} (как у прочих ошибок платформы)
//   - Swagger UI по пути /swagger с поддержкой выбора спецификации
//   - Регистрацию обработчиков через RegisterFn
//   - Auth Middleware
//   - Rate Limit Middleware
func WithOpenAPI(apiFS fs.FS, register RegisterFn) Option {
	return func(app *Application) error {
		app.apiFS = apiFS
		app.registerFn = register

		app.components.add(component(httpServer))

		if ok := app.components.add(component(openapiComponent)); !ok {
			return fmt.Errorf("%w: openapi", ErrComponentAlreadyExist)
		}

		return nil
	}
}

func initOpenAPI(ctx context.Context, app *Application) error {
	registerUUIDOnce.Do(func() {
		openapi3.DefineStringFormatValidator("uuid", openapi3.NewRegexpFormatValidator(uuidPattern))
	})

	specs, err := loadSpecsFromFS(app.apiFS)
	if err != nil {
		return fmt.Errorf("openapi: failed to load specs: %w", err)
	}

	if len(specs) == 0 {
		return fmt.Errorf("openapi: no YAML specs found in the provided filesystem")
	}

	specData := make([][]byte, 0, len(specs))
	specEntries := make([]swagger.SpecEntry, 0, len(specs))

	for _, s := range specs {
		specData = append(specData, s.Data)
		specEntries = append(specEntries, swagger.SpecEntry{Name: s.Name, Data: s.Data})
	}

	validationMw, err := httpValidationMiddleware(ctx, specData...)
	if err != nil {
		return fmt.Errorf("openapi validation init failed: %w", err)
	}

	swaggerMw, err := swagger.MultiSpecMiddleware(specEntries)
	if err != nil {
		return fmt.Errorf("openapi swagger init failed: %w", err)
	}

	app.middlewares.Add(swaggerMw)
	app.middlewares.Add(authMiddleware)
	app.middlewares.Add(rateLimitMiddleware)
	app.middlewares.Add(validationMw)

	if app.registerFn != nil {
		mux := http.NewServeMux()
		app.registerFn(ctx, mux)
		app.RegisterRouter(mux)
	}

	return nil
}

func loadSpecsFromFS(fsys fs.FS) ([]specFile, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	specs := make([]specFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !isYAMLFile(name) {
			continue
		}

		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", name, err)
		}

		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		specs = append(specs, specFile{Name: baseName, Data: data})
	}

	return specs, nil
}

func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))

	return ext == ".yaml" || ext == ".yml"
}
