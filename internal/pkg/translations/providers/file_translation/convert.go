package filetranslation

import "easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/translations"

func convertKeysToDictionary(dict map[string]map[string]string) translations.Dictionary {
	langsKeysMap := make(translations.Dictionary)
	for key, langs := range dict {
		for lang, text := range langs {
			if _, ok := langsKeysMap[lang]; !ok {
				langsKeysMap[lang] = make(map[string]string)
			}
			langsKeysMap[lang][key] = text
		}
	}
	return langsKeysMap
}
