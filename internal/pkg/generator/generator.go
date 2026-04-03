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

	"github.com/getkin/kin-openapi/openapi3"
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
		name := strings.TrimSuffix(filepath.Base(spec), filepath.Ext(spec))
		specNames = append(specNames, name)

		output := fmt.Sprintf("internal/api/%s/api.gen.go", name)
		if err := os.MkdirAll(filepath.Join(cfg.ServiceRoot, "internal/api/"+name), permRule); err != nil {
			return err
		}

		// oapi-codegen
		if err := RunOapiCodegen(cfg.ServiceRoot, spec, output, name); err != nil {
			return fmt.Errorf("oapi-codegen for %s: %w", name, err)
		}

		// scaffold (handler.go + register.gen.go)
		singleCfg := Config{
			SpecPath:    spec,
			GenPackage:  "internal/api/" + name,
			HandlerDir:  "internal/handler/" + name,
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
	exec.Command("go", "fmt", "./internal/handler/...", "./internal/api/...").Run()

	// CI-проверка
	if cfg.Check {
		return CICheck(cfg.ServiceRoot)
	}

	return nil
}

func newGenerator(cfg Config) (*Generator, error) {
	handlerName, err := deriveHandlerName(cfg.SpecPath)
	if err != nil {
		return nil, fmt.Errorf("failed to derive handler name from spec: %w", err)
	}

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
		handlerName: handlerName,
		methods:     methods,
		modulePath:  modulePath,
	}, nil
}

func (g *Generator) generate() error {
	handlerPath := filepath.Join(g.config.ServiceRoot, g.config.HandlerDir)
	genImportPath := g.modulePath + "/" + g.config.GenPackage
	pkgName := filepath.Base(g.config.HandlerDir)

	if err := generateHandlerFile(handlerPath, pkgName, g.handlerName, genImportPath, g.methods); err != nil {
		return fmt.Errorf("failed to generate handler: %w", err)
	}

	if err := generateRegisterFile(handlerPath, pkgName, g.handlerName, genImportPath); err != nil {
		return fmt.Errorf("failed to generate register: %w", err)
	}

	return nil
}

// deriveHandlerName извлекает имя структуры хэндлера из info.title спецификации.
// Например: "Items API" → "ItemsHandler", "User Management" → "UserManagementHandler".
// При отсутствии title используется имя файла.
func deriveHandlerName(specPath string) (string, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return "", fmt.Errorf("failed to load spec: %w", err)
	}

	title := ""
	if doc.Info != nil {
		title = doc.Info.Title
	}

	if title == "" {
		base := filepath.Base(specPath)
		title = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return cleanTitle(title) + "Handler", nil
}

func cleanTitle(title string) string {
	for _, suffix := range []string{" API", " api", " Service", " service", " Api"} {
		title = strings.TrimSuffix(title, suffix)
	}

	return toPascalCase(title)
}

func toPascalCase(s string) string {
	var result strings.Builder
	upper := true
	for _, r := range s {
		if r == ' ' || r == '-' || r == '_' {
			upper = true

			continue
		}
		if upper {
			result.WriteRune(unicode.ToUpper(r))
			upper = false
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
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
