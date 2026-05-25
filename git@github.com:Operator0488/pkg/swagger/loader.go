package swagger

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

var (
	DefaultPath = "api/openapi/api.swagger.json"
	rootMarker  = "go.mod"
)

func LoadOpenAPISpec(path string) ([]byte, error) {
	if path == "" {
		path = DefaultPath
	}

	if b, err := os.ReadFile(path); err == nil {
		return b, nil
	}

	if root, _ := findUpward(".", rootMarker); root != "" {
		if b, err := os.ReadFile(filepath.Join(root, path)); err == nil {
			return b, nil
		}
	}

	return nil, errors.New("openapi spec not found, set spec in api/openapi/api.swagger.json")
}

func findUpward(start, marker string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(abs, marker)); err == nil {
			return abs, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fs.ErrNotExist
		}
		abs = parent
	}
}
