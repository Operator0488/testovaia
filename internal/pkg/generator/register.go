package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

func generateRegisterFile(dir, pkg, genImport string) error {
	registerPath := filepath.Join(dir, "register.gen.go")

	if err := os.MkdirAll(dir, permRule); err != nil {
		return err
	}

	f, err := os.Create(registerPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Printf("CREATE: %s\n", registerPath)

	return registerTmpl.Execute(f, map[string]interface{}{
		"Package":   pkg,
		"GenImport": genImport,
	})
}
