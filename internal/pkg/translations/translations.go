package translations

import (
	"context"
	"fmt"
	"time"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/localize"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"golang.org/x/text/language"
)

// Dictionary переводы
//
//	{
//		"en": { "key": "translate", "key2": "translate 2" } },
//	    "fr": { "key": "translate" }
//	}
type Dictionary map[string]map[string]string

type Loader interface {
	GetTranslations(ctx context.Context, langs []string) (Dictionary, error)
}

type Uploader interface {
	UploadTranslations(context.Context, Dictionary) error
}

type TranslateManager interface {
	SetAvaialableLangs(langs []string)
	Load(context.Context, Loader) error
	Upload(context.Context, Loader, Uploader) error
	Watch(ctx context.Context, loader Loader, refresh time.Duration) context.CancelFunc
}

type translations struct {
	bundler localize.Bundler
	langs   []string
}

func NewManager(bundler localize.Bundler, langs []string) TranslateManager {
	return &translations{bundler: bundler, langs: langs}
}

// Load загружает в бандл переводы.
func (t *translations) Load(ctx context.Context, loader Loader) error {
	dictionary, err := loader.GetTranslations(ctx, t.langs)
	if err != nil {
		return err
	}
	for lang, messages := range dictionary {
		tag, err := language.Parse(lang)
		if err != nil {
			return fmt.Errorf("failed to parse lang to language.Tag, %w, lang: %s", err, lang)
		}
		if err := t.bundler.AddRawMessages(tag, messages); err != nil {
			return err
		}
	}
	return nil
}

// Upload загружает в провайдер новые переводы.
func (t *translations) Upload(ctx context.Context, from Loader, to Uploader) error {
	res, err := from.GetTranslations(ctx, t.langs)
	if err != nil {
		return err
	}
	if len(res) == 0 {
		return nil
	}

	return to.UploadTranslations(ctx, res)
}

// Watch обновляет переводы по таймеру.
func (t *translations) Watch(ctx context.Context, loader Loader, refresh time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)
	logCtx := logger.With(ctx, logger.String("component", "translations"))

	go func() {
		ticker := time.NewTicker(refresh)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := t.Load(ctx, loader); err != nil {
					logger.Error(logCtx, "failed to fetch translations", logger.Err(err))
				} else {
					logger.Info(logCtx, "translations refreshed")
				}
			}
		}
	}()

	return cancel
}

// SetAvaialableLangs обновление доступных языков.
func (t *translations) SetAvaialableLangs(langs []string) {
	t.langs = langs
}
