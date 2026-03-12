package localize

import (
	"context"

	"golang.org/x/text/language"
)

// contextKey - кастомный тип для хранения языка.
type contextKey string

const (
	langKey contextKey = "lang_tag_key"
)

// WithLocale создает контекст с указанным языком.
func WithLocale(ctx context.Context, lang string) (context.Context, error) {
	t, _, err := language.ParseAcceptLanguage(lang)
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, langKey, t), nil
}

// getLocaleFromContext извлекает язык которые был указан в контексте.
func getLocaleFromContext(ctx context.Context) ([]language.Tag, bool) {
	lang, ok := ctx.Value(langKey).([]language.Tag)
	return lang, ok
}
