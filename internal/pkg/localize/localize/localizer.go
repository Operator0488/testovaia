package localize

import (
	"context"
	"fmt"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/parser"
	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/plural"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"golang.org/x/text/language"
)

type localizer struct {
	bundle Bundler
}

type Localizer interface {
	// Get возвращает перевод по ключу, текущий язык определяется через контекст
	// Если перевод не найден или указан не валидный шаблон перевода то вернется сам ключ
	// Используйте localize.WithLocal(ctx, "en") чтобы установить свой язык
	Get(ctx context.Context, key string, opts ...Option) string

	// GetWithParams возвращает перевод по ключу, текущий язык определяется через контекст
	// Используйте localize.WithLocal(ctx, "en") чтобы установить свой язык
	GetWithParams(ctx context.Context, key string, params *LocalizeParams) string

	// TryGet возвращает перевод по ключу, текущий язык определяется через контекст
	// Если перевод не найден или указан не валидный шаблон перевода то вернется ошибка
	TryGet(ctx context.Context, key string, opts ...Option) (string, error)

	// TryGetWithParams возвращает перевод по ключу или ошибку, если не удалось получить перевод
	TryGetWithParams(ctx context.Context, key string, params *LocalizeParams) (string, error)
}

func NewLocalizer(bundle Bundler) Localizer {
	return &localizer{
		bundle: bundle,
	}
}

// LocalizeParams параметры которые можно задать для перевода.
type LocalizeParams struct {
	// TemplateData параметры которые используются в шаблоне перевода
	// Пример шаблона: Hello, {name}
	// Для того чтобы подставить в {name} имя, нужно добавить в TemplateData значение с ключом ["name"]="Mark"
	// После обработки такого шаблона мы получить строку Hello, Mark
	TemplateData map[string]any

	// PluralCount определяет, какая форма множественного числа сообщения используется.
	// Например: "У меня есть 1 кошка" или "У меня есть 10 кошек"
	// В данном случае если указать PluralCount=1
	// То вернется первый вариант, если указать 10 то - второй
	PluralCount any

	// Lang язык на который хотим перевести
	Lang string
}

var defaultTextParser = &parser.TextParser{}

type invalidPluralCountErr struct {
	messageID   string
	pluralCount interface{}
	err         error
}

func (e *invalidPluralCountErr) Error() string {
	return fmt.Sprintf("invalid plural count %#v for message id %q: %s", e.pluralCount, e.messageID, e.err)
}

type MessageNotFoundErr struct {
	Tag       language.Tag
	MessageID string
}

func (e *MessageNotFoundErr) Error() string {
	return fmt.Sprintf("message %q not found in language %q", e.MessageID, e.Tag)
}

// GetWithParams возвращает перевод по ключу, текущий язык определяется через контекст
// Используйте localize.WithLocal(ctx, "en") чтобы установить свой язык
func (l *localizer) GetWithParams(ctx context.Context, key string, lc *LocalizeParams) string {
	translate, tag, err := l.localizeWithTag(ctx, key, lc)
	if err != nil {
		logger.Warn(ctx, "translation failed",
			logger.String("component", "localize"),
			logger.String("translation_key", key),
			logger.Err(err),
		)

		l.bundle.AddMessages(tag, &Message{
			ID:    key,
			Other: key,
		})
		return key
	}
	return translate
}

// Get возвращает перевод по ключу, текущий язык определяется через контекст
// Если перевод не найден то вернется сам ключ
// Используйте localize.WithLocal(ctx, "en") чтобы установить свой язык
func (l *localizer) Get(ctx context.Context, key string, opts ...Option) string {
	lc := &LocalizeParams{}
	for _, o := range opts {
		if o != nil {
			o(lc)
		}
	}
	return l.GetWithParams(ctx, key, lc)
}

func (l *localizer) TryGetWithParams(ctx context.Context, key string, params *LocalizeParams) (string, error) {
	key, _, err := l.localizeWithTag(ctx, key, params)
	return key, err
}

func (l *localizer) TryGet(ctx context.Context, key string, opts ...Option) (string, error) {
	lc := &LocalizeParams{}
	for _, o := range opts {
		if o != nil {
			o(lc)
		}
	}
	key, _, err := l.localizeWithTag(ctx, key, lc)
	return key, err
}

func (l *localizer) getLangTag(ctx context.Context, lc *LocalizeParams) (language.Tag, error) {
	if len(lc.Lang) == 0 {
		return l.bundle.GetTagFromContext(ctx), nil
	}
	return language.Parse(lc.Lang)
}

func (l *localizer) localizeWithTag(ctx context.Context, key string, lc *LocalizeParams) (string, language.Tag, error) {
	tag, err := l.getLangTag(ctx, lc)
	if err != nil {
		return key, tag, err
	}

	messageID := key
	var operands *plural.Operands
	templateData := lc.TemplateData
	if lc.PluralCount != nil {
		var err error
		operands, err = plural.NewOperands(lc.PluralCount)
		if err != nil {
			return "", tag, &invalidPluralCountErr{messageID: messageID, pluralCount: lc.PluralCount, err: err}
		}
		if templateData == nil {
			templateData = make(map[string]any, 1)
		}
		// хардкод для tolgee, в нем именованный параметр value используется для плюралки
		templateData["value"] = lc.PluralCount
		templateData["PluralCount"] = lc.PluralCount
	}

	pluralForm, template, err := l.bundle.GetMessageTemplate(ctx, tag, messageID, operands)
	if template == nil {
		return "", tag, err
	}

	msg, err2 := template.execute(pluralForm, templateData, defaultTextParser)
	if err2 != nil {
		if err == nil {
			err = err2
		}

		// Фолбэк если не нашли других форм
		if pluralForm != plural.Other {
			msg2, err3 := template.execute(plural.Other, templateData, defaultTextParser)
			if err3 == nil {
				msg = msg2
			}
		}
	}
	return msg, tag, err
}
