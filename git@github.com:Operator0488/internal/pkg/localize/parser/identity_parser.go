package parser

// IdentityParser который просто возвращает строку, так как  в этой строке нет шаблонов
type IdentityParser struct{}

func (IdentityParser) Parse(src, leftDelim, rightDelim string) (ParsedTemplate, error) {
	return &identityParsedTemplate{src: src}, nil
}

type identityParsedTemplate struct {
	src string
}

func (t *identityParsedTemplate) Execute(data any) (string, error) {
	return t.src, nil
}
