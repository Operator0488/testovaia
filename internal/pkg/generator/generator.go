package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
)

const permRule = 0o755

type Config struct {
	SpecPath    string
	GenPackage  string
	HandlerDir  string
	ServiceRoot string
}

type Generator struct {
	config      Config
	handlerName string
	methods     []string
	modulePath  string
}

func NewGenerator(cfg Config) (*Generator, error) {
	handlerName, err := deriveHandlerName(cfg.SpecPath)
	if err != nil {
		return nil, fmt.Errorf("failed to derive handler name from spec: %w", err)
	}

	genFilePath := filepath.Join(cfg.ServiceRoot, cfg.GenPackage, "api.gen.go")
	methods, err := parseStrictInterface(genFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse StrictServerInterface from %s: %w", genFilePath, err)
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

func (g *Generator) Generate() error {
	handlerPath := filepath.Join(g.config.ServiceRoot, g.config.HandlerDir)
	genImportPath := g.modulePath + "/" + g.config.GenPackage
	pkgName := filepath.Base(g.config.HandlerDir)

	if err := generateHandlerFile(handlerPath, pkgName, g.handlerName, g.methods, genImportPath); err != nil {
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

// parseStrictInterface парсит сгенерированный файл и извлекает имена методов
// из интерфейса StrictServerInterface.
func parseStrictInterface(filePath string) ([]string, error) {
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
			if !ok || typeSpec.Name.Name != "StrictServerInterface" {
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
		return nil, fmt.Errorf("StrictServerInterface not found in %s", filePath)
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

var handlerTmpl = template.Must(template.New("handler").Parse(`package {{.Package}}

import (
	"context"

	gen "{{.GenImport}}"
)

// {{.HandlerName}} реализует gen.StrictServerInterface.
type {{.HandlerName}} struct{}
{{range .Methods}}
func (h *{{$.HandlerName}}) {{.}}(ctx context.Context, req gen.{{.}}RequestObject) (gen.{{.}}ResponseObject, error) {
	// todo: implement me
}
{{end}}`))

func generateHandlerFile(dir, pkg, handlerName string, methods []string, genImport string) error {
	handlerPath := filepath.Join(dir, "handler.go")

	if _, err := os.Stat(handlerPath); err == nil {
		existing, parseErr := findExistingMethods(handlerPath, handlerName)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "WARNING: could not parse existing %s: %v\n", handlerPath, parseErr)

			return nil
		}

		existingSet := make(map[string]struct{}, len(existing))
		for _, e := range existing {
			existingSet[e] = struct{}{}
		}

		var missing []string
		for _, m := range methods {
			if _, ok := existingSet[m]; !ok {
				missing = append(missing, m)
			}
		}

		if len(missing) > 0 {
			fmt.Fprintf(os.Stderr, "WARNING: %s is missing methods: %s\n", handlerPath, strings.Join(missing, ", "))
			fmt.Fprintln(os.Stderr, "Add them manually or delete the file and re-run the scaffold.")
		} else {
			fmt.Printf("SKIP: %s already exists and implements all methods\n", handlerPath)
		}

		return nil
	}

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

var registerTmpl = template.Must(template.New("register").Parse(`package {{.Package}}

import (
	"context"
	"net/http"

	gen "{{.GenImport}}"
)

// Register регистрирует все HTTP-обработчики на mux.
func Register(ctx context.Context, mux *http.ServeMux) {
	h := &{{.HandlerName}}{}
	gen.HandlerFromMux(gen.NewStrictHandler(h, nil), mux)
}
`))

func generateRegisterFile(dir, pkg, handlerName string, genImport string) error {
	registerPath := filepath.Join(dir, "register.go")

	if _, err := os.Stat(registerPath); err == nil {
		fmt.Printf("SKIP: %s already exists\n", registerPath)

		return nil
	}

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
		"Package":     pkg,
		"HandlerName": handlerName,
		"GenImport":   genImport,
	})
}

func findExistingMethods(filePath, handlerName string) ([]string, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filePath, nil, 0)
	if err != nil {
		return nil, err
	}

	var methods []string
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			continue
		}
		for _, field := range funcDecl.Recv.List {
			if starExpr, ok := field.Type.(*ast.StarExpr); ok {
				if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == handlerName {
					methods = append(methods, funcDecl.Name.Name)
				}
			}
		}
	}

	return methods, nil
}
