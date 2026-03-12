## Компонент локолизации

Компонент доступен через интерфейс `localize.Localizer`

### Примеры использования:

```go
// Получение простого перевода по ключу
func (f *someType) SomeExampleFunc(ctx context.Context) {
    str:=f.localize.Get(ctx, "translate.key")
}
```

```go
// Получение перевода с использованием шаблона
// В данном примере используется шаблон: Hello friend, {name}!
func (f *someType) SomeExampleFunc(ctx context.Context) {
    str := t.localizer.Get(ctx, "translate.key", localize.WithValue("name", name))
}
```

```go
// Получение перевода для различных форм множественного числа
// Например: "У меня есть 1 кошка" или "У меня есть 10 кошек"
func (f *someType) SomeExampleFunc(ctx context.Context) {
    str := t.localizer.Get(ctx, 
        "translate.key",
		localize.WithPlural(count),
	)
}
```

```go
// Установка в контекст нового языка
func (f *someType) LanguageMiddleware(ctx context.Context, lang string) {
    ctx, err := localize.WithLocale(ctx, lang)
}
```

