package swagger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustNewHandler(t *testing.T, specs []SpecEntry, externalPrefix string) *multiSpecHandler {
	t.Helper()
	h, err := newMultiSpecHandler(specs, externalPrefix)
	require.NoError(t, err)

	return h
}

func minimalYAML(name string) SpecEntry {
	return SpecEntry{
		Name: name,
		Data: []byte(`openapi: "3.0.0"
info:
  title: ` + name + `
  version: "1.0.0"
paths: {}`),
	}
}

func TestNewMultiSpecHandler_BasePath(t *testing.T) {
	tests := []struct {
		name           string
		externalPrefix string
		wantBasePath   string
	}{
		{
			name:           "с префиксом",
			externalPrefix: "/api/some-service",
			wantBasePath:   "/api/some-service/swagger",
		},
		{
			name:           "без префикса",
			externalPrefix: "",
			wantBasePath:   "/swagger",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := mustNewHandler(t, []SpecEntry{minimalYAML("comments")}, tt.externalPrefix)
			assert.Equal(t, tt.wantBasePath, h.basePath)
		})
	}
}

func TestNewMultiSpecHandler_ConvertsYAMLtoJSON(t *testing.T) {
	h := mustNewHandler(t, []SpecEntry{minimalYAML("comments")}, "")

	require.Len(t, h.jsonSpecs, 1)

	var parsed map[string]any
	err := json.Unmarshal(h.jsonSpecs[0].Data, &parsed)
	assert.NoError(t, err)
}

func TestNewMultiSpecHandler_InvalidYAML(t *testing.T) {
	_, err := newMultiSpecHandler([]SpecEntry{
		{Name: "bad", Data: []byte(`:: invalid yaml ::`)},
	}, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad")
}

func TestPatchServers(t *testing.T) {
	spec := []byte(`{"openapi":"3.0.0","paths":{"/v1/customers":{}}}`)

	tests := []struct {
		name           string
		externalPrefix string
		wantServerURL  string
	}{
		{
			name:           "подставляет externalPrefix в servers",
			externalPrefix: "/api/some-service",
			wantServerURL:  "/api/some-service",
		},
		{
			name:           "пустой префикс — servers становится пустой строкой",
			externalPrefix: "",
			wantServerURL:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := mustNewHandler(t, []SpecEntry{minimalYAML("comments")}, tt.externalPrefix)

			result := h.patchServers(spec)

			var got map[string]any
			require.NoError(t, json.Unmarshal(result, &got))

			servers, ok := got["servers"].([]any)
			require.True(t, ok, "servers должен быть массивом")
			require.Len(t, servers, 1)

			first, ok := servers[0].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.wantServerURL, first["url"])
		})
	}
}

func TestPatchServers_InvalidJSON_ReturnsOriginal(t *testing.T) {
	h := mustNewHandler(t, []SpecEntry{minimalYAML("comments")}, "/api/some-service")

	original := []byte(`not a json`)
	result := h.patchServers(original)

	assert.Equal(t, original, result)
}

func TestServe_Routing(t *testing.T) {
	specs := []SpecEntry{minimalYAML("comments"), minimalYAML("customers")}
	h := mustNewHandler(t, specs, "/api/some-service")
	notFound := http.HandlerFunc(http.NotFound)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantHeader map[string]string
	}{
		{
			name:       "redirect без trailing slash",
			path:       "/api/some-service/swagger",
			wantStatus: http.StatusMovedPermanently,
			wantHeader: map[string]string{
				"Location": "/api/some-service/swagger/",
			},
		},
		{
			name:       "UI страница",
			path:       "/api/some-service/swagger/",
			wantStatus: http.StatusOK,
			wantHeader: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			},
		},
		{
			name:       "спецификация comments",
			path:       "/api/some-service/swagger/comments/openapi.json",
			wantStatus: http.StatusOK,
			wantHeader: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:       "спецификация customers",
			path:       "/api/some-service/swagger/customers/openapi.json",
			wantStatus: http.StatusOK,
			wantHeader: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:       "несуществующий путь — проваливается в next",
			path:       "/api/some-service/v1/customers",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			h.serve(w, r, notFound)

			assert.Equal(t, tt.wantStatus, w.Code)
			for k, v := range tt.wantHeader {
				assert.Equal(t, v, w.Header().Get(k))
			}
		})
	}
}

func TestServe_SingleSpec_CompatPath(t *testing.T) {
	// для одной спеки работает /swagger/openapi.json
	h := mustNewHandler(t, []SpecEntry{minimalYAML("comments")}, "/api/some-service")
	notFound := http.HandlerFunc(http.NotFound)

	r := httptest.NewRequest(http.MethodGet, "/api/some-service/swagger/openapi.json", nil)
	w := httptest.NewRecorder()

	h.serve(w, r, notFound)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestServe_SpecContainsPatchedServers(t *testing.T) {
	h := mustNewHandler(t, []SpecEntry{minimalYAML("comments")}, "/api/some-service")
	notFound := http.HandlerFunc(http.NotFound)

	r := httptest.NewRequest(http.MethodGet, "/api/some-service/swagger/comments/openapi.json", nil)
	w := httptest.NewRecorder()

	h.serve(w, r, notFound)

	require.Equal(t, http.StatusOK, w.Code)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &spec))

	servers, ok := spec["servers"].([]any)
	require.True(t, ok)
	require.Len(t, servers, 1)
	assert.Equal(t, "/api/some-service", servers[0].(map[string]any)["url"])
}

func TestServe_UIContainsRelativeURLs(t *testing.T) {
	specs := []SpecEntry{minimalYAML("comments"), minimalYAML("customers")}
	h := mustNewHandler(t, specs, "/api/some-service")
	notFound := http.HandlerFunc(http.NotFound)

	r := httptest.NewRequest(http.MethodGet, "/api/some-service/swagger/", nil)
	w := httptest.NewRecorder()

	h.serve(w, r, notFound)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `"./comments/openapi.json"`)
	assert.Contains(t, body, `"./customers/openapi.json"`)
	assert.NotContains(t, body, `"/swagger/`)
	assert.NotContains(t, body, `"/api/`)
}
