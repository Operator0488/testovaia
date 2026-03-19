package main

import (
	"flag"
	"fmt"
	"os"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/generator"
)

func main() {
	cfg := generator.Config{}

	flag.StringVar(&cfg.SpecPath, "spec", "", "Path to OpenAPI spec file")
	flag.StringVar(&cfg.GenPackage, "gen-package", "", "Relative path to generated package")
	flag.StringVar(&cfg.HandlerDir, "handler-dir", "", "Relative path to handler directory")
	flag.StringVar(&cfg.ServiceRoot, "service-root", "", "Service root directory")

	flag.Parse()

	if cfg.SpecPath == "" || cfg.GenPackage == "" || cfg.HandlerDir == "" || cfg.ServiceRoot == "" {
		flag.Usage()
		os.Exit(1)
	}

	gen, err := generator.NewGenerator(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openapi-scaffold: %v\n", err)
		os.Exit(1)
	}

	if err := gen.Generate(); err != nil {
		fmt.Fprintf(os.Stderr, "openapi-scaffold: %v\n", err)
		os.Exit(1)
	}
}
