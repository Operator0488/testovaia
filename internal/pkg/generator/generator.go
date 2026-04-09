package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

const permRule = 0o755

type Generator struct {
	config      Config
	handlerName string
	methods     []string
	modulePath  string
}

func RunSingle(cfg Config) error {
	batchCfg := Config{
		APIDir:      filepath.Dir(cfg.SpecPath),
		ServiceRoot: cfg.ServiceRoot,
		Check:       cfg.Check,
	}

	return RunBatch(batchCfg)
}

func RunBatch(cfg Config) error {
	modulePath, err := getModulePath(cfg.ServiceRoot)
	if err != nil {
		return err
	}

	specs, err := DiscoverSpecs(cfg.APIDir)
	if err != nil {
		return err
	}

	specNames := make([]string, 0, len(specs))
	for _, spec := range specs {
		packageName := derivePackageName(spec)
		specNames = append(specNames, packageName)

		output := fmt.Sprintf("internal/api/%s/api.gen.go", packageName)
		if err := os.MkdirAll(filepath.Join(cfg.ServiceRoot, "internal/api/"+packageName), permRule); err != nil {
			return err
		}

		// oapi-codegen
		if err := RunOapiCodegen(cfg.ServiceRoot, spec, output, packageName); err != nil {
			return fmt.Errorf("oapi-codegen for %s: %w", packageName, err)
		}

		// scaffold
		singleCfg := Config{
			SpecPath:    spec,
			GenPackage:  "internal/api/" + packageName,
			ServiceRoot: cfg.ServiceRoot,
		}
		gen, err := newGenerator(singleCfg)
		if err != nil {
			return err
		}
		if err := gen.generate(); err != nil {
			return err
		}
	}

	// Агрегатор
	if err := GenerateAggregator(cfg.ServiceRoot, modulePath, specNames); err != nil {
		return err
	}

	// go fmt
	exec.Command("go", "fmt", "./internal/api/...").Run()

	// CI-проверка
	if cfg.Check {
		return CICheck(cfg.ServiceRoot)
	}

	return nil
}

func newGenerator(cfg Config) (*Generator, error) {
	genFilePath := filepath.Join(cfg.ServiceRoot, cfg.GenPackage, "api.gen.go")
	methods, err := parseServerInterface(genFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ServerInterface from %s: %w", genFilePath, err)
	}

	modulePath, err := getModulePath(cfg.ServiceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read module path: %w", err)
	}

	return &Generator{
		config:      cfg,
		handlerName: deriveHandlerName(cfg.SpecPath),
		methods:     methods,
		modulePath:  modulePath,
	}, nil
}

func (g *Generator) generate() error {
	genPath := filepath.Join(g.config.ServiceRoot, g.config.GenPackage)
	genImportPath := g.modulePath + "/" + g.config.GenPackage
	pkgName := filepath.Base(g.config.GenPackage)

	if err := generateHandlerFile(genPath, pkgName, g.handlerName, genImportPath, g.methods); err != nil {
		return fmt.Errorf("failed to generate handler: %w", err)
	}

	if err := generateRegisterFile(genPath, pkgName, g.handlerName, genImportPath); err != nil {
		return fmt.Errorf("failed to generate register: %w", err)
	}

	return nil
}

func deriveBaseName(spec string) string {
	base := filepath.Base(spec)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	if name == "" {
		return ""
	}

	runes := []rune(name)
	lastIdx := len(runes) - 1
	if runes[lastIdx] == 's' || runes[lastIdx] == 'S' {
		runes = runes[:lastIdx]
	}

	return string(runes)
}

func derivePackageName(spec string) string {
	return deriveBaseName(spec)
}

func deriveHandlerName(specPath string) string {
	name := deriveBaseName(specPath)
	if name == "" {
		return "Handler"
	}

	runes := []rune(name)
	runes[0] = unicode.ToUpper(runes[0])

	return string(runes) + "Handler"
}

// parseServerInterface парсит сгенерированный файл и извлекает имена методов
// из интерфейса ServerInterface (std-http-server).
func parseServerInterface(filePath string) ([]string, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filePath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	methods := make([]string, 0)
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "ServerInterface" {
				continue
			}
			iType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			for _, method := range iType.Methods.List {
				if len(method.Names) > 0 {
					methods = append(methods, method.Names[0].Name)
				}
			}
		}
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("ServerInterface not found in %s", filePath)
	}

	return methods, nil
}

func getModulePath(serviceRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(serviceRoot, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", fmt.Errorf("module path not found in go.mod")
}
