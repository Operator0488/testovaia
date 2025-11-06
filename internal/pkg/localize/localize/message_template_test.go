package localize

import (
	"reflect"
	"testing"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/plural"
)

func TestMessageTemplate(t *testing.T) {
	mt := newMessageTemplate(&Message{ID: "HelloWorld", Other: "Hello World"}, "", "")
	if mt.PluralTemplates[plural.Other].Src != "Hello World" {
		t.Fatal(mt.PluralTemplates)
	}
}

func TestNilMessageTemplate(t *testing.T) {
	if mt := newMessageTemplate(&Message{ID: "HelloWorld"}, "", ""); mt != nil {
		t.Fatal(mt)
	}
}

func TestMessageTemplatePluralFormMissing(t *testing.T) {
	mt := newMessageTemplate(&Message{ID: "HelloWorld", Other: "Hello World"}, "", "")
	s, err := mt.Execute(plural.Few, nil)
	if s != "" {
		t.Errorf("expected %q; got %q", "", s)
	}
	expectedErr := pluralFormNotFoundError{pluralForm: plural.Few, messageID: "HelloWorld"}
	if !reflect.DeepEqual(err, expectedErr) {
		t.Errorf("expected error %#v; got %#v", expectedErr, err)
	}
}
