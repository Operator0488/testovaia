package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

func generateHandlerFile(dir, pkg, handlerName, genImport string, methods []string) error {
	handlerPath := filepath.Join(dir, "handler.gen.go")

	if err := os.MkdirAll(dir, permRule); err != nil {
		return err
	}

	f, err := os.Create(handlerPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Printf("CREATE: %s (handler: %s)\n", handlerPath, handlerName)

	return handlerTmpl.Execute(f, map[string]interface{}{
		"Package":     pkg,
		"HandlerName": handlerName,
		"Methods":     methods,
		"GenImport":   genImport,
	})
}
