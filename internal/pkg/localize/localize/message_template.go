package localize

import (
	"fmt"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/parser"
	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/plural"
)

type messageTemplate struct {
	*Message
	PluralTemplates map[plural.Form]*template
}

func newMessageTemplate(m *Message, leftDelim, rightDelim string) *messageTemplate {
	pluralTemplates := map[plural.Form]*template{}
	setPluralTemplate(pluralTemplates, plural.Zero, m.Zero, leftDelim, rightDelim)
	setPluralTemplate(pluralTemplates, plural.One, m.One, leftDelim, rightDelim)
	setPluralTemplate(pluralTemplates, plural.Two, m.Two, leftDelim, rightDelim)
	setPluralTemplate(pluralTemplates, plural.Few, m.Few, leftDelim, rightDelim)
	setPluralTemplate(pluralTemplates, plural.Many, m.Many, leftDelim, rightDelim)
	setPluralTemplate(pluralTemplates, plural.Other, m.Other, leftDelim, rightDelim)
	if len(pluralTemplates) == 0 {
		return nil
	}
	return &messageTemplate{
		Message:         m,
		PluralTemplates: pluralTemplates,
	}
}

func setPluralTemplate(pluralTemplates map[plural.Form]*template, pluralForm plural.Form, src, leftDelim, rightDelim string) {
	if src != "" {
		pluralTemplates[pluralForm] = &template{
			Src:        src,
			LeftDelim:  leftDelim,
			RightDelim: rightDelim,
		}
	}
}

type pluralFormNotFoundError struct {
	pluralForm plural.Form
	messageID  string
}

func (e pluralFormNotFoundError) Error() string {
	return fmt.Sprintf("message %q has no plural form %q", e.messageID, e.pluralForm)
}

func (mt *messageTemplate) Execute(pluralForm plural.Form, data interface{}) (string, error) {
	t := mt.PluralTemplates[pluralForm]
	if t == nil {
		return "", pluralFormNotFoundError{
			pluralForm: pluralForm,
			messageID:  mt.ID,
		}
	}
	parser := &parser.TextParser{}
	return t.Execute(parser, data)
}

func (mt *messageTemplate) execute(pluralForm plural.Form, data interface{}, parser parser.Parser) (string, error) {
	t := mt.PluralTemplates[pluralForm]
	if t == nil {
		return "", pluralFormNotFoundError{
			pluralForm: pluralForm,
			messageID:  mt.ID,
		}
	}
	return t.Execute(parser, data)
}
