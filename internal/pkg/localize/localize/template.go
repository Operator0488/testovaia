package localize

import (
	"sync"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/parser"
)

type template struct {
	Src        string
	LeftDelim  string
	RightDelim string

	parseOnce      sync.Once
	parsedTemplate parser.ParsedTemplate
	parseError     error
}

func (t *template) Execute(p parser.Parser, data interface{}) (string, error) {
	var pt parser.ParsedTemplate
	var err error
	t.parseOnce.Do(func() {
		t.parsedTemplate, t.parseError = p.Parse(t.Src, t.LeftDelim, t.RightDelim)
	})
	pt, err = t.parsedTemplate, t.parseError

	if err != nil {
		return "", err
	}
	return pt.Execute(data)
}
