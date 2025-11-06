package localize

import (
	"context"
	"fmt"
	"sync"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/plural"
	"golang.org/x/text/language"
)

type bundle struct {
	defaultLanguage  language.Tag
	messageTemplates map[language.Tag]map[string]*messageTemplate
	pluralRules      plural.Rules
	tags             []language.Tag
	matcher          language.Matcher
	leftDelim        string
	rightDelim       string
	mu               *sync.RWMutex
}

// Bundler управляет справочником переводов.
type Bundler interface {
	// AddRawMessages добавляет в справочник новый словарь для языка.
	AddRawMessages(tag language.Tag, messages map[string]string) error

	// AddMessages добавляет в справочник новый словарь для языка.
	AddMessages(tag language.Tag, messages ...*Message) error

	// GetMessageTemplate возвращает шаблон для ключа для формирования финальной строки.
	GetMessageTemplate(ctx context.Context, tag language.Tag, id string, operands *plural.Operands) (plural.Form, *messageTemplate, error)

	// GetTagFromContext получить текущий язык из контекста
	GetTagFromContext(ctx context.Context) language.Tag
}

var artTag = language.MustParse("art")

// NewBundle конструктор для создания справочника.
func NewBundle(defaultLanguage language.Tag) Bundler {
	b := &bundle{
		defaultLanguage: defaultLanguage,
		pluralRules:     plural.DefaultRules(),
		leftDelim:       "{",
		rightDelim:      "}",
		mu:              &sync.RWMutex{},
	}
	b.pluralRules[artTag] = b.pluralRules.Rule(language.English)
	b.addTag(defaultLanguage)
	return b
}

func (b *bundle) SetDelim(leftDelim, rightDelim string) {
	if leftDelim != "" && rightDelim != "" {
		b.leftDelim = leftDelim
		b.rightDelim = rightDelim
	}
}

// AddMessages добавляет словарь для указанного языка предварительно распарсив пришедшее сообщение.
func (b *bundle) AddRawMessages(tag language.Tag, messages map[string]string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	pluralRule := b.pluralRules.Rule(tag)
	if pluralRule == nil {
		return fmt.Errorf("no plural rule registered for %s", tag)
	}

	if b.messageTemplates == nil {
		b.messageTemplates = map[language.Tag]map[string]*messageTemplate{}
	}
	if b.messageTemplates[tag] == nil {
		b.messageTemplates[tag] = map[string]*messageTemplate{}
		b.addTag(tag)
	}
	for key, icuMessage := range messages {
		m, err := parseICU(key, icuMessage)
		if err != nil {
			return err
		}
		b.messageTemplates[tag][m.ID] = newMessageTemplate(m, b.leftDelim, b.rightDelim)
	}
	return nil
}

// AddMessages добавляет словарь для указанного языка.
func (b *bundle) AddMessages(tag language.Tag, messages ...*Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	pluralRule := b.pluralRules.Rule(tag)
	if pluralRule == nil {
		return fmt.Errorf("no plural rule registered for %s", tag)
	}
	if b.messageTemplates == nil {
		b.messageTemplates = map[language.Tag]map[string]*messageTemplate{}
	}
	if b.messageTemplates[tag] == nil {
		b.messageTemplates[tag] = map[string]*messageTemplate{}
		b.addTag(tag)
	}
	for _, m := range messages {
		b.messageTemplates[tag][m.ID] = newMessageTemplate(m, b.leftDelim, b.rightDelim)
	}
	return nil
}

func (b *bundle) addTag(tag language.Tag) {
	for _, t := range b.tags {
		if t == tag {
			// Язык уже есть
			return
		}
	}
	b.tags = append(b.tags, tag)
	b.matcher = language.NewMatcher(b.tags)
}

func (b *bundle) getMessage(tag language.Tag, id string) *messageTemplate {
	b.mu.RLock()
	defer b.mu.RUnlock()

	templates := b.messageTemplates[tag]
	if templates == nil {
		return nil
	}
	return templates[id]
}

func (l *bundle) GetTagFromContext(ctx context.Context) language.Tag {
	tags, ok := getLocaleFromContext(ctx)
	if !ok {
		return l.defaultLanguage
	}
	_, i, _ := l.matcher.Match(tags...)
	tag := l.tags[i]
	return tag
}

func (l *bundle) GetMessageTemplate(ctx context.Context, tag language.Tag, id string, operands *plural.Operands) (plural.Form, *messageTemplate, error) {
	mt := l.getMessage(tag, id)
	pluralForm := l.pluralForm(tag, operands)
	if mt != nil {
		return pluralForm, mt, nil
	}

	// если тег был дефолтный то смысла дальше искать нет
	if tag == l.defaultLanguage {
		return plural.Invalid, nil, &MessageNotFoundErr{Tag: tag, MessageID: id}
	}

	// пробуем использовать по дефолтному тегу
	mt = l.getMessage(l.defaultLanguage, id)
	if mt != nil {
		return pluralForm, mt, nil
	}

	return plural.Invalid, nil, &MessageNotFoundErr{Tag: tag, MessageID: id}
}

func (l *bundle) pluralForm(tag language.Tag, operands *plural.Operands) plural.Form {
	if operands == nil {
		return plural.Other
	}
	return l.pluralRules.Rule(tag).PluralFormFunc(operands)
}
