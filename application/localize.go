package application

import (
	"context"
	"fmt"
	"os"
	"time"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/config"
	"git.vepay.dev/knoknok/backend-platform/internal/pkg/localize/localize"
	"git.vepay.dev/knoknok/backend-platform/internal/pkg/translations"
	filetranslation "git.vepay.dev/knoknok/backend-platform/internal/pkg/translations/providers/file_translation"
	"git.vepay.dev/knoknok/backend-platform/internal/pkg/translations/providers/tolgee"
	"git.vepay.dev/knoknok/backend-platform/pkg/di"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"golang.org/x/text/language"
)

var (
	localizeComponent = NewComponent("localize", initLocalizeClient, Noop)
)

const (
	localizePath              = "./bootstrap"
	localizeFileName          = "localize.json"
	defaultTranslationRefresh = 10 * time.Minute
)

// WithLocalize add localize component
func WithLocalize() Option {
	return func(app *Application) error {
		app.components.add(component(localizeComponent))
		return nil
	}
}

func initLocalizeClient(ctx context.Context, app *Application) error {
	// local provider
	readProvider := filetranslation.NewProvider(os.DirFS(localizePath), localizeFileName)

	availableLangs := []string{"en"}
	bundler := localize.NewBundle(language.English)
	app.translateManager = translations.NewManager(bundler, availableLangs)

	// загрузка данных из файла
	if err := app.translateManager.Load(ctx, readProvider); err != nil {
		return fmt.Errorf("failed to bootstrap translations, %w", err)
	}

	app.Localizer = localize.NewLocalizer(bundler)
	di.Register(ctx, app.Localizer)

	initLocalizeTolgee(ctx, app, readProvider)

	return nil
}

// initLocalizeTolgee запуск провайдера tolgee
func initLocalizeTolgee(
	ctx context.Context,
	app *Application,
	fromProvider translations.Loader) {
	tConfig := config.NewConfigWatcher("tolgee", app.Env, tolgee.NewConfig)
	if err := tConfig.Get().Validate(); err != nil {
		if err == tolgee.ErrDisabled {
			logger.Info(ctx, "tolgee translations provider disabled")
		} else {
			logger.Error(ctx, "tolgee config is not valid", logger.Err(err))
		}
	}

	app.Env.Subscribe(tConfig)

	// установка доступных языков
	tolgeeClient := tolgee.NewClient(tConfig)

	if langs := getAvailableLangs(ctx, tolgeeClient); len(langs) > 0 {
		app.translateManager.SetAvaialableLangs(langs)
	}

	tolgeeProvider := tolgee.NewProvider(tolgeeClient)

	// загрузка новых данных в tolgee, приложение не останавливаем
	if err := app.translateManager.Upload(ctx, fromProvider, tolgeeProvider); err != nil {
		logger.Error(ctx, "failed to upload translations to provider", logger.Err(err))
	}

	// загрузка ключей из tolgee
	if err := app.translateManager.Load(ctx, tolgeeProvider); err != nil {
		logger.Error(ctx, "failed to download translations from provider", logger.Err(err))
	}

	runTranslationsWatcher(ctx, app, tolgeeProvider)
}

// getAvailableLangs доступные в tolgee языки
func getAvailableLangs(
	ctx context.Context,
	tolgeeClient *tolgee.Client,
) []string {

	langs, err := tolgeeClient.GetLanguages(ctx)
	if err != nil {
		logger.Error(ctx, "failed to load languages from tolgee", logger.Err(err))
		return []string{}
	} else {
		availableLangs := make([]string, 0, len(langs))
		for _, lang := range langs {
			availableLangs = append(availableLangs, lang.Tag)
		}
		return availableLangs
	}
}

// runTranslationsWatcher запускает вотчер для слежения за апдейтами переводов
func runTranslationsWatcher(
	ctx context.Context,
	app *Application,
	loader translations.Loader,
) {
	interval := app.Env.GetDuration("localize.refresh_duration")
	if interval == 0 {
		interval = defaultTranslationRefresh
	}

	cancel := app.translateManager.Watch(ctx, loader, interval)
	app.Closer.Add(func() error {
		cancel()
		return nil
	})
}
