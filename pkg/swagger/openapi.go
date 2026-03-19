package swagger

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

//go:embed multi_index.html
var multiUIHTML []byte

// SpecEntry описывает одну OpenAPI спецификацию с именем
type SpecEntry struct {
	Name string
	Data []byte
}

type multiSpecHandler struct {
	basePath  string
	jsonSpecs []SpecEntry
	uiPage    []byte
}

// MultiSpecMiddleware обслуживает Swagger UI для нескольких OpenAPI спецификаций.
// Каждая спецификация доступна по пути /swagger/{name}/openapi.json.
// UI показывает выпадающий список для выбора спецификации.
// Для одной спецификации также работает обратно-совместимый путь {basePath}/openapi.json.
func MultiSpecMiddleware(specs []SpecEntry) (func(http.HandlerFunc) http.HandlerFunc, error) {
	handler, err := newMultiSpecHandler(specs, swaggerPath)
	if err != nil {
		return nil, err
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			handler.serve(w, r, next)
		}
	}, nil
}

func newMultiSpecHandler(specs []SpecEntry, basePath string) (*multiSpecHandler, error) {
	jsonSpecs := make([]SpecEntry, 0, len(specs))
	for _, s := range specs {
		j, err := ensureJSON(s.Data)
		if err != nil {
			return nil, fmt.Errorf("swagger: failed to convert spec %q to JSON: %w", s.Name, err)
		}
		jsonSpecs = append(jsonSpecs, SpecEntry{Name: s.Name, Data: j})
	}

	uiPage, err := renderMultiSpecUI(specs, basePath)
	if err != nil {
		return nil, fmt.Errorf("swagger: failed to render UI: %w", err)
	}

	return &multiSpecHandler{basePath: basePath, jsonSpecs: jsonSpecs, uiPage: uiPage}, nil
}

func (h *multiSpecHandler) serve(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	p := r.URL.Path

	switch {
	case p == h.basePath:
		http.Redirect(w, r, h.basePath+"/", http.StatusMovedPermanently)
	case p == h.basePath+"/":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		_, _ = w.Write(h.uiPage)
	case p == h.basePath+"/openapi.json" && len(h.jsonSpecs) == 1:
		serveJSONSpec(w, h.jsonSpecs[0].Data)
	default:
		if spec, ok := h.findSpec(p); ok {
			serveJSONSpec(w, spec.Data)

			return
		}
		next(w, r)
	}
}

func (h *multiSpecHandler) findSpec(path string) (SpecEntry, bool) {
	for _, spec := range h.jsonSpecs {
		if path == h.basePath+"/"+spec.Name+"/openapi.json" {
			return spec, true
		}
	}

	return SpecEntry{}, false
}

func serveJSONSpec(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	_, _ = w.Write(data)
}

func renderMultiSpecUI(specs []SpecEntry, basePath string) ([]byte, error) {
	type urlEntry struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	}

	urls := make([]urlEntry, 0, len(specs))
	for _, s := range specs {
		urls = append(urls, urlEntry{
			URL:  basePath + "/" + s.Name + "/openapi.json",
			Name: s.Name,
		})
	}
	sort.Slice(urls, func(i, j int) bool { return urls[i].Name < urls[j].Name })

	urlsJSON, err := json.Marshal(urls)
	if err != nil {
		return nil, err
	}

	result := bytes.Replace(multiUIHTML, []byte("__SWAGGER_URLS__"), urlsJSON, -1)
	if bytes.Equal(result, multiUIHTML) {
		return nil, fmt.Errorf("swagger: placeholder __SWAGGER_URLS__ not found in multi_index.html")
	}

	return result, nil
}

func ensureJSON(spec []byte) ([]byte, error) {
	trimmed := strings.TrimSpace(string(spec))
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return spec, nil
	}

	var data any
	if err := yaml.Unmarshal(spec, &data); err != nil {
		return nil, err
	}

	return json.Marshal(data)
}
