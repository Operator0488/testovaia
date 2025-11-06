package swagger

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"net/http"
	"reflect"
	"time"
)

const swaggerPath = "/swagger"

//go:embed index.html
var uiHTML []byte

type Registrar func(ctx context.Context, mux *runtime.ServeMux) error

type Manager struct {
	spec []byte
	regs []Registrar
}

func New(spec []byte) *Manager {
	m := &Manager{spec: spec}
	return m
}

func (m *Manager) Add(r Registrar) {
	m.regs = append(m.regs, r)
}

func (m *Manager) Middleware(ctx context.Context) (func(http.HandlerFunc) http.HandlerFunc, error) {

	gw := runtime.NewServeMux()
	for _, r := range m.regs {
		if err := r(ctx, gw); err != nil {
			return nil, err
		}
	}

	spec := m.spec

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case swaggerPath:
				http.Redirect(w, r, swaggerPath+"/", http.StatusMovedPermanently)
				return
			case swaggerPath + "/":
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				_, _ = w.Write(uiHTML)
				return
			case swaggerPath + "/openapi.json":
				if len(spec) == 0 {
					http.Error(w, "openapi spec is empty", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "public, max-age=60")
				w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
				_, _ = w.Write(spec)
				return
			}

			cw := &captureWriter{
				ResponseWriter: w,
				header:         http.Header{},
			}
			gw.ServeHTTP(cw, r)

			if cw.status == 0 || cw.status == http.StatusNotFound {
				next(w, r)
				return
			}

			copyHeader(w.Header(), cw.header)
			if cw.status != 0 {
				w.WriteHeader(cw.status)
			}

			_, _ = w.Write(cw.body.Bytes())
		}
	}, nil
}

// Gateway — RegisterXxxHandlerServer и сервер
func Gateway(reg any, srv any) Registrar {
	v := reflect.ValueOf(reg)
	t := v.Type()

	return func(ctx context.Context, mux *runtime.ServeMux) error {
		argSrv := reflect.ValueOf(srv)
		want := t.In(2)

		if want.Kind() == reflect.Interface && argSrv.Type().Implements(want) {
			argSrv = argSrv.Convert(want)
		} else {
			return fmt.Errorf("swagger.Gateway: server %v does not implement %v", argSrv.Type(), want)
		}
		out := v.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(mux), argSrv})
		if err, _ := out[0].Interface().(error); err != nil {
			return err
		}
		return nil
	}
}

type captureWriter struct {
	http.ResponseWriter
	status int
	header http.Header
	body   bytes.Buffer
	wroteH bool
}

func (w *captureWriter) Header() http.Header { return w.header }

func (w *captureWriter) WriteHeader(code int) {
	if !w.wroteH {
		w.wroteH, w.status = true, code
	}
}

func (w *captureWriter) Write(p []byte) (int, error) {
	if !w.wroteH {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(p)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		dst.Del(k)
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
