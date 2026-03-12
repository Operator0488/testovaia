package parser

type Parser interface {
	// Parse парсит шаблон и возвращает объект ParsedTemplate,
	// который можно запустить через Execute и передать туда доп. данные для шаблона
	Parse(src, leftDelim, rightDelim string) (ParsedTemplate, error)
}

// ParsedTemplate исполняемый шаблон
type ParsedTemplate interface {
	// Execute applies a parsed template to the specified data.
	Execute(data any) (string, error)
}
