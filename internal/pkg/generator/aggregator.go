package generator

import (
	"os"
	"path/filepath"
)

// GenerateAggregator создаёт internal/api/register.gen.go
func GenerateAggregator(serviceRoot, modulePath string, specNames []string) error {
	dir := filepath.Join(serviceRoot, "internal", "api")
	if err := os.MkdirAll(dir, permRule); err != nil {
		return err
	}

	outPath := filepath.Join(dir, "register.gen.go")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return aggregatorTmpl.Execute(f, map[string]any{
		"ModulePath": modulePath,
		"Names":      specNames,
	})
}
