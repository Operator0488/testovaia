package main

import (
	"flag"
	"fmt"
	"os"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/generator"
)

func main() {
	cfg := generator.Config{}

	flag.StringVar(&cfg.SpecPath, "spec", "", "Path to single OpenAPI spec")
	flag.StringVar(&cfg.GenPackage, "gen-package", "", "Relative path to generated package")
	flag.StringVar(&cfg.ServiceRoot, "service-root", "", "Service root directory")
	flag.StringVar(&cfg.APIDir, "api-dir", "", "Directory with OpenAPI specs (batch mode)")
	flag.BoolVar(&cfg.Check, "check", false, "CI mode: verify files are up to date")
	flag.Parse()

	if cfg.SpecPath != "" {
		err := generator.RunSingle(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "openapi-scaffold: %v\n", err)
			os.Exit(1)
		}

		return
	}

	if cfg.APIDir != "" {
		err := generator.RunBatch(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "openapi-scaffold: %v\n", err)
			os.Exit(1)
		}

		return
	}

	flag.Usage()
	os.Exit(1)
}
