package filetranslation

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/translations"
)

type FileProvider interface {
	translations.Loader

	Close()
}

type fileProvider struct {
	dir      fs.FS
	fileName string
	// небольшая оптимизация чтобы заново не загружать файл
	cache translations.Dictionary
}

// NewProvider создает провайдер для работы локальным файлом.
func NewProvider(dir fs.FS, fileName string) FileProvider {
	return &fileProvider{dir: dir, fileName: fileName}
}

// GetTranslations implements FileProvider.
func (f *fileProvider) GetTranslations(ctx context.Context, _ []string) (translations.Dictionary, error) {
	if f.cache != nil {
		return f.cache, nil
	}
	file, err := f.dir.Open(f.fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var result map[string]map[string]string
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return make(translations.Dictionary), nil
	}

	dict := convertKeysToDictionary(result)

	f.cache = dict

	return dict, nil
}

func (f *fileProvider) Close() {
	f.cache = nil
}
