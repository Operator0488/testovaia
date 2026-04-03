package generator

import (
	"fmt"
	"path/filepath"
)

// DiscoverSpecs находит все *.yaml/*.yml в директории
func DiscoverSpecs(apiDir string) ([]string, error) {
	var specs []string
	for _, pattern := range []string{"*.yaml", "*.yml"} {
		matches, err := filepath.Glob(filepath.Join(apiDir, pattern))
		if err != nil {
			return nil, err
		}
		specs = append(specs, matches...)
	}

	if len(specs) == 0 {
		return nil, fmt.Errorf("no OpenAPI specs found in %s", apiDir)
	}

	return specs, nil
}
