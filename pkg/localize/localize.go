package localize

import (
	"context"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/localize"
)

type Localizer = localize.Localizer

type Plurable = localize.Plurable

type Option = localize.Option

// WithPlural определяет, какая форма множественного числа сообщения используется.
// Например: "У меня есть 1 кошка" или "У меня есть 10 кошек"
// В данном случае если указать PluralCount=1
// То вернется первый вариант, если указать 10 то - второй
func WithPlural[T localize.Plurable](plural T) Option {
	return localize.WithPlural(plural)
}

// WithData предоставляет словарь с данными, которые используются внутри шаблона
// Например: Добро пожаловать, {username}!
// Для такого шаблона нужно использовать WithData(map[string]any{"username": "Петр"})
func WithData(data map[string]any) Option {
	return localize.WithData(data)
}

// WithValue предоставляет словарь с данными, которые используются внутри шаблона
// Например: Добро пожаловать, {username}!
// Для такого шаблона нужно использовать WithValue("username", "Петр")
func WithValue(key string, value any) Option {
	return localize.WithValue(key, value)
}

// WithLang устанавливает в параметрах язык на который нужно перевести ключ.
func WithLang(lang string) Option {
	return localize.WithLang(lang)
}

// WithLocale Добавление в контекст текущего языка пользователя.
func WithLocale(ctx context.Context, lang string) (context.Context, error) {
	return localize.WithLocale(ctx, lang)
}
