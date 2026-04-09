package generator

import (
	"fmt"
	"os"
	"os/exec"
)

// CICheck запускает git diff и возвращает ошибку если файлы устарели
func CICheck(serviceRoot string) error {
	cmd := exec.Command("git", "diff", "--quiet", "--exit-code", "internal/api/")
	cmd.Dir = serviceRoot
	if err := cmd.Run(); err != nil {
		stat := exec.Command("git", "diff", "--stat")
		stat.Dir = serviceRoot
		stat.Stdout = os.Stderr
		stat.Run()

		return fmt.Errorf("generated files are stale, run 'make generate' and commit")
	}

	return nil
}
