package localize

import (
	"strings"
	"testing"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/parser"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		template *template
		parser   parser.Parser
		data     interface{}
		result   string
		err      string
		noallocs bool
	}{
		{
			template: &template{
				Src:        "hello",
				LeftDelim:  "{",
				RightDelim: "}",
			},
			result:   "hello",
			noallocs: true,
		},
		{
			template: &template{
				Src:        "hello {.Noun}",
				LeftDelim:  "{",
				RightDelim: "}",
			},
			data: map[string]string{
				"Noun": "world",
			},
			result: "hello world",
		},
		{
			template: &template{
				Src:        "hello {",
				LeftDelim:  "{",
				RightDelim: "}",
			},
			err:      "unclosed action",
			noallocs: true,
		},
	}

	for _, test := range tests {
		t.Run(test.template.Src, func(t *testing.T) {
			if test.parser == nil {
				test.parser = &parser.TextParser{}
			}
			result, err := test.template.Execute(test.parser, test.data)
			if actual := str(err); !strings.Contains(str(err), test.err) {
				t.Errorf("expected err %q to contain %q", actual, test.err)
			}
			if result != test.result {
				t.Errorf("expected result %q; got %q", test.result, result)
			}
			allocs := testing.AllocsPerRun(10, func() {
				_, _ = test.template.Execute(test.parser, test.data)
			})
			if test.noallocs && allocs > 0 {
				t.Errorf("expected no allocations; got %f", allocs)
			}
		})
	}
}

func str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
