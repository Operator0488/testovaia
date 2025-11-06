package tolgee

import (
	"context"
	"maps"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/translations"
)

type TolgeeProvider interface {
	translations.Loader
	translations.Uploader
}

type tolgeeProvider struct {
	client *Client
}

// NewProvider создает провайдер для работы с Tolgee.
func NewProvider(client *Client) TolgeeProvider {
	return &tolgeeProvider{client: client}
}

// GetTranslations implements TolgeeProvider.
func (t *tolgeeProvider) GetTranslations(ctx context.Context, langs []string) (translations.Dictionary, error) {
	res, err := t.client.GetTranslations(ctx, langs)
	if err != nil {
		return nil, err
	}
	dict := make(translations.Dictionary, len(res))
	maps.Copy(dict, res)
	return dict, nil
}

// UploadTranslations implements TolgeeProvider.
func (t *tolgeeProvider) UploadTranslations(ctx context.Context, dict translations.Dictionary) error {
	keys := convertDictionaryToImportKeys(dict, t.client.GetTags())
	return t.client.ImportKeys(ctx, keys)
}
