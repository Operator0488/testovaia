package tolgee

import "git.vepay.dev/knoknok/backend-platform/internal/pkg/translations"

func convertDictionaryToImportKeys(dict translations.Dictionary, tags []string) []ImportKey {
	keysMap := make(map[string]ImportKey, 0)
	for lang, messages := range dict {
		for key, text := range messages {
			var importKey ImportKey

			importKey, ok := keysMap[key]
			if !ok {
				importKey = ImportKey{Name: key, Tags: tags, Translations: make(map[string]string)}
			}
			importKey.Translations[lang] = text
			keysMap[key] = importKey
		}
	}

	res := make([]ImportKey, 0, len(keysMap))
	for _, t := range keysMap {
		res = append(res, t)
	}
	return res
}
