package response

import (
	"bytes"
	"encoding/json"
	"net/http"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/middleware"
)

// envelopeWriter перехватывает запись ответа для обёртки в Envelope
type envelopeWriter struct {
	http.ResponseWriter

	status int
	header http.Header
	body   bytes.Buffer
	wrote  bool
}

func (w *envelopeWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}

	return w.header
}

func (w *envelopeWriter) WriteHeader(code int) {
	if !w.wrote {
		w.wrote = true
		w.status = code
	}
}

func (w *envelopeWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}

	return w.body.Write(p)
}

func (w *envelopeWriter) flush() {
	// копируем заголовки (кроме Content-Length — пересчитаем)
	for k, vv := range w.header {
		if k == "Content-Length" {
			continue
		}
		for _, v := range vv {
			w.ResponseWriter.Header().Add(k, v)
		}
	}

	if w.status == 0 {
		w.status = http.StatusOK
	}

	body := w.body.Bytes()

	// 2xx — оборачиваем payload в envelope
	if w.status >= 200 && w.status < 300 {
		var payload any
		if len(body) > 0 {
			_ = json.Unmarshal(body, &payload)
		}
		env := Envelope{Payload: payload}
		w.ResponseWriter.WriteHeader(w.status)
		_ = json.NewEncoder(w.ResponseWriter).Encode(env)

		return
	}

	// 4xx/5xx — body уже в формате Envelope от WriteError
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = w.ResponseWriter.Write(body)
}

// EnvelopeMiddleware оборачивает успешные (2xx) JSON-ответы в Envelope.
// Служебные пути (/healthz, /metrics, /swagger) пропускаются без изменений.
// Ошибки (4xx/5xx) ожидаются в формате Envelope от response.WriteError.
func EnvelopeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if middleware.IsInfraPath(r.URL.Path) {
			next(w, r)

			return
		}

		ew := &envelopeWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next(ew, r)
		ew.flush()
	}
}
