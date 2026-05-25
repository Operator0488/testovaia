package generator

import (
	"os"
	"os/exec"
)

const oapiCodegenTool = "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest"

// RunOapiCodegen запускает oapi-codegen для одного спека.
func RunOapiCodegen(serviceRoot, specPath, outputPath, pkgName string) error {
	cmd := exec.Command("go", "run", oapiCodegenTool,
		"--package", pkgName,
		"--generate", "models,std-http-server",
		"--o", outputPath,
		specPath)
	cmd.Dir = serviceRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
