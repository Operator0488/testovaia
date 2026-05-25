package generator

import (
	"embed"
	"fmt"
	"os"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var (
	aggregatorTmpl = mustLoad("templates/aggregator.tmpl")
	handlerTmpl    = mustLoad("templates/handler.tmpl")
	registerTmpl   = mustLoad("templates/register.tmpl")
)

func mustLoad(path string) *template.Template {
	data, err := templateFS.ReadFile(path)
	if err != nil {
		fmt.Errorf("failed to load template %s: %w", path, err)
		os.Exit(1)
	}

	return template.Must(template.New(path).Parse(string(data)))
}
