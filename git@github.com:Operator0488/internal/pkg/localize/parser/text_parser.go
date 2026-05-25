package parser

import (
	"bytes"
	"errors"
	"strings"
	"text/template"
)

type TextParser struct {
	Option string
}

func (te *TextParser) Parse(src, leftDelim, rightDelim string) (ParsedTemplate, error) {
	if leftDelim == "" || rightDelim == "" {
		return nil, errors.New("left and right delim is required")
	}

	if !strings.Contains(src, leftDelim) {
		// Fast path to avoid parsing a template that has no actions.
		return &identityParsedTemplate{src: src}, nil
	}

	option := "missingkey=default"
	if te.Option != "" {
		option = te.Option
	}

	tmpl, err := template.New("").Delims(leftDelim, rightDelim).Option(option).Parse(src)
	if err != nil {
		return nil, err
	}
	return &parsedTextTemplate{tmpl: tmpl}, nil
}

type parsedTextTemplate struct {
	tmpl *template.Template
}

func (t *parsedTextTemplate) Execute(data any) (string, error) {
	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
