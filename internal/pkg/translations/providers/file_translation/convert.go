package filetranslation

import "git.vepay.dev/knoknok/backend-platform/internal/pkg/translations"

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
