package localize

import (
	"context"
	"fmt"
	"testing"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/plural"
	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
)

type localizerTest struct {
	name              string
	defaultLanguage   language.Tag
	messages          map[language.Tag][]*Message
	dirtyMessages     map[language.Tag]map[string]string
	acceptLang        string
	conf              *LocalizeParams
	translateKey      string
	expectedErr       error
	expectedLocalized string
}

func localizerTests() []localizerTest {
	return []localizerTest{
		{
			name:            "message id not mismatched",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.English: {{ID: "HelloWorld", Other: "Hello!"}},
			},
			acceptLang:        "en",
			translateKey:      "HelloWorld",
			conf:              &LocalizeParams{},
			expectedLocalized: "Hello!",
		},
		{
			name:              "missing translation from default language",
			defaultLanguage:   language.English,
			acceptLang:        "en",
			translateKey:      "HelloWorld",
			conf:              &LocalizeParams{},
			expectedErr:       &MessageNotFoundErr{Tag: language.English, MessageID: "HelloWorld"},
			expectedLocalized: "",
		},
		{
			name:            "empty translation without fallback",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.English: {{ID: "HelloWorld", Other: "Hello World!"}},
				language.Spanish: {{ID: "HelloWorld"}},
			},
			acceptLang:        "en",
			translateKey:      "HelloWorld",
			conf:              &LocalizeParams{},
			expectedLocalized: "Hello World!",
		},
		{
			name:            "missing translation from default language with other translation",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.Spanish: {{ID: "HelloWorld", Other: "other"}},
			},
			acceptLang:   "en",
			translateKey: "HelloWorld",
			conf:         &LocalizeParams{},
			expectedErr:  &MessageNotFoundErr{Tag: language.English, MessageID: "HelloWorld"},
		},
		{
			name:              "missing translations from not default language",
			defaultLanguage:   language.English,
			acceptLang:        "en",
			translateKey:      "HelloWorld",
			conf:              &LocalizeParams{},
			expectedErr:       &MessageNotFoundErr{Tag: language.English, MessageID: "HelloWorld"},
			expectedLocalized: "",
		},
		{
			name:            "missing translation not default language with other translation",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.French:  {{ID: "HelloWorld", Other: "other"}},
				language.Spanish: {{ID: "SomethingElse", Other: "other"}},
			},
			acceptLang:        "en",
			translateKey:      "HelloWorld",
			conf:              &LocalizeParams{},
			expectedErr:       &MessageNotFoundErr{Tag: language.English, MessageID: "HelloWorld"},
			expectedLocalized: "",
		},
		{
			name:            "plural count one, bundle message",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:    "Cats",
					One:   "I have {.value} cat",
					Other: "I have {.value} cats",
				}},
			},
			acceptLang:   "en",
			translateKey: "Cats",
			conf: &LocalizeParams{
				PluralCount: 1,
			},
			expectedLocalized: "I have 1 cat",
		},
		{
			name:            "plural count other, bundle message",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:    "Cats",
					One:   "I have {.value} cat",
					Other: "I have {.value} cats",
				}},
			},
			acceptLang:   "en",
			translateKey: "Cats",
			conf: &LocalizeParams{
				PluralCount: 2,
			},
			expectedLocalized: "I have 2 cats",
		},
		{
			name:            "plural count float, bundle message",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:    "Cats",
					One:   "I have {.value} cat",
					Other: "I have {.value} cats",
				}},
			},
			acceptLang:   "en",
			translateKey: "Cats",
			conf: &LocalizeParams{
				PluralCount: "2.5",
			},
			expectedLocalized: "I have 2.5 cats",
		},
		{
			name:            "plural count missing other, default message",
			defaultLanguage: language.English,
			acceptLang:      "en",
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:  "Cats",
					One: "I have {.value} cat",
				}},
			},
			translateKey: "Cats",
			conf: &LocalizeParams{
				PluralCount: 2,
			},
			expectedErr: &pluralFormNotFoundError{messageID: "Cats", pluralForm: plural.Other},
		},
		{
			name:            "plural count float, default message",
			defaultLanguage: language.English,
			acceptLang:      "en",
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:    "Cats",
					One:   "I have {.value} cat",
					Other: "I have {.value} cats",
				}},
			},
			translateKey: "Cats",
			conf: &LocalizeParams{
				PluralCount: "2.5",
			},
			expectedLocalized: "I have 2.5 cats",
		},
		{
			name:            "template data, bundle message",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:    "HelloPerson",
					Other: "Hello {.Person}",
				}},
			},
			acceptLang:   "en",
			translateKey: "HelloPerson",
			conf: &LocalizeParams{
				TemplateData: map[string]any{
					"Person": "Nick",
				},
			},
			expectedLocalized: "Hello Nick",
		},
		{
			name:            "template data, plural count one, bundle message",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:    "PersonCats",
					One:   "{.Person} has {.value} cat",
					Other: "{.Person} has {.value} cats",
				}},
			},
			acceptLang:   "en",
			translateKey: "PersonCats",
			conf: &LocalizeParams{
				TemplateData: map[string]interface{}{
					"Person": "Nick",
				},
				PluralCount: 1,
			},
			expectedLocalized: "Nick has 1 cat",
		},
		{
			name:            "template data, plural count other, bundle message",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:    "PersonCats",
					One:   "{.Person} has {.value} cat",
					Other: "{.Person} has {.value} cats",
				}},
			},
			acceptLang:   "en",
			translateKey: "PersonCats",
			conf: &LocalizeParams{
				TemplateData: map[string]interface{}{
					"Person": "Nick",
				},
				PluralCount: 2,
			},
			expectedLocalized: "Nick has 2 cats",
		},
		{
			name:            "template data, plural count float, bundle message",
			defaultLanguage: language.English,
			translateKey:    "PersonCats",
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:    "PersonCats",
					One:   "{.Person} has {.value} cat",
					Other: "{.Person} has {.value} cats",
				}},
			},
			acceptLang: "en",
			conf: &LocalizeParams{
				TemplateData: map[string]interface{}{
					"Person": "Nick",
				},
				PluralCount: "2.5",
			},
			expectedLocalized: "Nick has 2.5 cats",
		},
		{
			name:            "no fallback",
			defaultLanguage: language.Spanish,
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:    "Hello",
					Other: "Hello!",
				}},
				language.French: {{
					ID:    "Goodbye",
					Other: "Goodbye!",
				}},
			},
			acceptLang:   "fr",
			translateKey: "Hello",
			conf:         &LocalizeParams{},
			expectedErr:  &MessageNotFoundErr{Tag: language.French, MessageID: "Hello"},
		},
	}
}

func localizerDirtyTests() []localizerTest {
	return []localizerTest{
		{
			name:            "message id not mismatched",
			defaultLanguage: language.English,
			dirtyMessages: map[language.Tag]map[string]string{
				language.English: {
					"HelloWorld": "Hello!",
				},
			},
			acceptLang:        "en",
			translateKey:      "HelloWorld",
			conf:              &LocalizeParams{},
			expectedLocalized: "Hello!",
		},
		{
			name:            "empty translation without fallback",
			defaultLanguage: language.English,
			dirtyMessages: map[language.Tag]map[string]string{
				language.English: {
					"HelloWorld": "Hello World!",
				},
				language.Spanish: {"HelloWorld": ""},
			},
			acceptLang:        "en",
			translateKey:      "HelloWorld",
			conf:              &LocalizeParams{},
			expectedLocalized: "Hello World!",
		},
		{
			name:            "plural count one, bundle message",
			defaultLanguage: language.English,
			messages: map[language.Tag][]*Message{
				language.English: {{
					ID:    "Cats",
					One:   "",
					Other: "",
				}},
			},
			dirtyMessages: map[language.Tag]map[string]string{
				language.English: {
					"Cats": "{value, plural,one {I have # cat} other {I have # cats}}",
				},
				language.Spanish: {"HelloWorld": ""},
			},
			acceptLang:   "en",
			translateKey: "Cats",
			conf: &LocalizeParams{
				PluralCount: 1,
			},
			expectedLocalized: "I have 1 cat",
		},
		{
			name:            "plural count other, bundle message",
			defaultLanguage: language.English,
			dirtyMessages: map[language.Tag]map[string]string{
				language.English: {
					"Cats": "{value, plural,one {I have # cat} other {I have # cats}}",
				},
				language.Spanish: {"HelloWorld": ""},
			},
			acceptLang:   "en",
			translateKey: "Cats",
			conf: &LocalizeParams{
				PluralCount: 2,
			},
			expectedLocalized: "I have 2 cats",
		},
		{
			name:            "plural count float, bundle message",
			defaultLanguage: language.English,
			dirtyMessages: map[language.Tag]map[string]string{
				language.English: {
					"Cats": "{value, plural,one {I have # cat} other {I have # cats}}",
				},
				language.Spanish: {"HelloWorld": ""},
			},
			acceptLang:   "en",
			translateKey: "Cats",
			conf: &LocalizeParams{
				PluralCount: "2.5",
			},
			expectedLocalized: "I have 2.5 cats",
		},
		{
			name:            "plural count missing other, default message",
			defaultLanguage: language.English,
			acceptLang:      "en",
			dirtyMessages: map[language.Tag]map[string]string{
				language.English: {
					"Cats": "{value, plural,one {I have # cat}}",
				},
				language.Spanish: {"HelloWorld": ""},
			},
			translateKey: "Cats",
			conf: &LocalizeParams{
				PluralCount: 2,
			},
			expectedErr: &pluralFormNotFoundError{messageID: "Cats", pluralForm: plural.Other},
		},
		{
			name:            "template data, bundle message",
			defaultLanguage: language.English,
			dirtyMessages: map[language.Tag]map[string]string{
				language.English: {
					"HelloPerson": "Hello {Person}",
				},
				language.Spanish: {"HelloWorld": ""},
			},
			acceptLang:   "en",
			translateKey: "HelloPerson",
			conf: &LocalizeParams{
				TemplateData: map[string]any{
					"Person": "Nick",
				},
			},
			expectedLocalized: "Hello Nick",
		},
		{
			name:            "template data, plural count one, bundle message",
			defaultLanguage: language.English,
			dirtyMessages: map[language.Tag]map[string]string{
				language.English: {
					"PersonCats": "{count, plural,one {{Person} has # cat} other {{Person} has # cats}}",
				},
				language.Spanish: {"HelloWorld": ""},
			},
			acceptLang:   "en",
			translateKey: "PersonCats",
			conf: &LocalizeParams{
				TemplateData: map[string]interface{}{
					"Person": "Nick",
					"count":  1,
				},
				PluralCount: 1,
			},
			expectedLocalized: "Nick has 1 cat",
		},
		{
			name:            "template data, plural count other, bundle message",
			defaultLanguage: language.English,
			dirtyMessages: map[language.Tag]map[string]string{
				language.English: {
					"PersonCats": "{count, plural,one {{Person} has # cat} other {{Person} has # cats}}",
				},
				language.Spanish: {"HelloWorld": ""},
			},
			acceptLang:   "en",
			translateKey: "PersonCats",
			conf: &LocalizeParams{
				TemplateData: map[string]interface{}{
					"Person": "Nick",
					"count":  2,
				},
				PluralCount: 2,
			},
			expectedLocalized: "Nick has 2 cats",
		},
		{
			name:            "template data, plural count float, bundle message",
			defaultLanguage: language.English,
			dirtyMessages: map[language.Tag]map[string]string{
				language.English: {
					"PersonCats": "{value, plural,one {{Person} has # cat} other {{Person} has # cats}}",
				},
				language.Spanish: {"HelloWorld": ""},
			},
			acceptLang:   "en",
			translateKey: "PersonCats",
			conf: &LocalizeParams{
				TemplateData: map[string]interface{}{
					"Person": "Nick",
				},
				PluralCount: "2.5",
			},
			expectedLocalized: "Nick has 2.5 cats",
		},
	}
}

func TestLocalizer_TryGetWithParams(t *testing.T) {
	for _, test := range localizerTests() {
		t.Run(test.name, func(t *testing.T) {
			bundle := NewBundle(test.defaultLanguage)
			for tag, messages := range test.messages {
				if err := bundle.AddMessages(tag, messages...); err != nil {
					t.Fatal(err)
				}
			}
			check := func(localized string, err error) {
				t.Helper()
				if (err != nil && test.expectedErr == nil) || (err == nil && test.expectedErr != nil) {
					t.Errorf("\nexpected error: %#v\n     got error: %#v", test.expectedErr, err)
				}
				if err != nil && test.expectedErr != nil && !assert.EqualError(t, err, test.expectedErr.Error()) {
					t.Errorf("\nexpected error: %#v\n     got error: %#v", test.expectedErr, err)
				}
				if localized != test.expectedLocalized {
					t.Errorf("expected localized string %q; got %q", test.expectedLocalized, localized)
				}
			}
			localizer := NewLocalizer(bundle)
			ctx, err := WithLocale(context.Background(), test.acceptLang)
			assert.NoError(t, err)
			check(localizer.TryGetWithParams(ctx, test.translateKey, test.conf))
		})
	}
}

func TestLocalizer_TryGet(t *testing.T) {
	for _, test := range localizerTests() {
		t.Run(test.name, func(t *testing.T) {
			bundle := NewBundle(test.defaultLanguage)
			for tag, messages := range test.messages {
				if err := bundle.AddMessages(tag, messages...); err != nil {
					t.Fatal(err)
				}
			}
			check := func(localized string, err error) {
				t.Helper()
				if (err != nil && test.expectedErr == nil) || (err == nil && test.expectedErr != nil) {
					t.Errorf("\nexpected error: %#v\n     got error: %#v", test.expectedErr, err)
				}
				if err != nil && test.expectedErr != nil && !assert.EqualError(t, err, test.expectedErr.Error()) {
					t.Errorf("\nexpected error: %#v\n     got error: %#v", test.expectedErr, err)
				}
				if localized != test.expectedLocalized {
					t.Errorf("expected localized string %q; got %q", test.expectedLocalized, localized)
				}
			}
			localizer := NewLocalizer(bundle)
			ctx, err := WithLocale(context.Background(), test.acceptLang)
			assert.NoError(t, err)

			var pluralCount Option
			if test.conf.PluralCount != nil {
				switch v := test.conf.PluralCount.(type) {
				case int:
					pluralCount = WithPlural(v)
				case string:
					pluralCount = WithPlural(v)
				}
			}

			templateData := WithData(test.conf.TemplateData)

			str, err := localizer.TryGet(ctx, test.translateKey,
				templateData,
				pluralCount,
			)
			check(str, err)
		})
	}
}

func TestLocalizer_LocalizeDirtyMessages(t *testing.T) {
	for _, test := range localizerDirtyTests() {
		t.Run(test.name, func(t *testing.T) {
			bundle := NewBundle(test.defaultLanguage)

			for tag, messages := range test.dirtyMessages {
				if err := bundle.AddRawMessages(tag, messages); err != nil {
					t.Fatal(err)
				}
			}
			check := func(localized string, err error) {
				t.Helper()
				if (err != nil && test.expectedErr == nil) || (err == nil && test.expectedErr != nil) {
					t.Errorf("\nexpected error: %#v\n     got error: %#v", test.expectedErr, err)
				}
				if err != nil && test.expectedErr != nil && !assert.EqualError(t, err, test.expectedErr.Error()) {
					t.Errorf("\nexpected error: %#v\n     got error: %#v", test.expectedErr, err)
				}
				if localized != test.expectedLocalized {
					t.Errorf("expected localized string %q; got %q", test.expectedLocalized, localized)
				}
			}
			localizer := NewLocalizer(bundle)
			ctx, err := WithLocale(context.Background(), test.acceptLang)
			assert.NoError(t, err)

			check(localizer.TryGetWithParams(ctx, test.translateKey, test.conf))
		})
	}
}

func TestMessageNotFoundError(t *testing.T) {
	actual := (&MessageNotFoundErr{Tag: language.AmericanEnglish, MessageID: "hello"}).Error()
	expected := `message "hello" not found in language "en-US"`
	if actual != expected {
		t.Fatalf("expected %q; got %q", expected, actual)
	}
}

func TestInvalidPluralCountError(t *testing.T) {
	actual := (&invalidPluralCountErr{messageID: "hello", pluralCount: "blah", err: fmt.Errorf("error")}).Error()
	expected := `invalid plural count "blah" for message id "hello": error`
	if actual != expected {
		t.Fatalf("expected %q; got %q", expected, actual)
	}
}
