package localize

type Plurable interface {
	int | int8 | int16 | int32 | int64 | string
}

type Option func(*LocalizeParams)

// WithPlural определяет, какая форма множественного числа сообщения используется.
// Например: "У меня есть 1 кошка" или "У меня есть 10 кошек"
// В данном случае если указать PluralCount=1
// То вернется первый вариант, если указать 10 то - второй
func WithPlural[T Plurable](plural T) Option {
	return func(lc *LocalizeParams) {
		lc.PluralCount = plural
	}
}

// WithData предоставляет словарь с данными, которые используются внутри шаблона
// Например: Добро пожаловать, {username}!
// Для такого шаблона нужно использовать WithData(map[string]any{"username": "Петр"})
func WithData(data map[string]any) Option {
	return func(lc *LocalizeParams) {
		lc.TemplateData = data
	}
}

// WithValue предоставляет словарь с данными, которые используются внутри шаблона
// Например: Добро пожаловать, {username}!
// Для такого шаблона нужно использовать WithValue("username", "Петр")
func WithValue(key string, value any) Option {
	return func(lc *LocalizeParams) {
		if lc.TemplateData == nil {
			lc.TemplateData = make(map[string]any, 1)
		}
		lc.TemplateData[key] = value
	}
}

// WithLang устанавливает в параметрах язык на который нужно перевести ключ.
func WithLang(lang string) Option {
	return func(lc *LocalizeParams) {
		lc.Lang = lang
	}
}
