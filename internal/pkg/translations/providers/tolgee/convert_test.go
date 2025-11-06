package tolgee

import (
	"testing"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/translations"
)

func TestConvertDictionaryToImportKeys(t *testing.T) {
	// Входные данные
	input := translations.Dictionary{
		"en": {
			"Backend.ServiceApp.Submit.Error":   "Failed to submit",
			"Backend.ServiceApp.Submit.Success": "Successfully submitted",
			"Backend.ServiceApp.HelloFriend":    "Hello {name}!",
		},
		"fr": {
			"Backend.ServiceApp.Submit.Error":   "Échec de la soumission",
			"Backend.ServiceApp.Submit.Success": "Soumis avec succès",
			"Backend.ServiceApp.HelloFriend":    "Bonjorn {name}!",
		},
		"de": {
			"Backend.ServiceApp.Submit.Error":   "Übermittlung fehlgeschlagen",
			"Backend.ServiceApp.Submit.Success": "Erfolgreich übermittelt",
		},
	}

	// Вызываем функцию
	result := convertDictionaryToImportKeys(input, []string{})

	// Проверяем общее количество ключей
	expectedKeyCount := 3
	if len(result) != expectedKeyCount {
		t.Errorf("Expected %d keys, got %d", expectedKeyCount, len(result))
	}

	// Создаем мапу для удобства проверки
	resultMap := make(map[string]ImportKey)
	for _, key := range result {
		resultMap[key.Name] = key
	}

	// Проверяем ключ Backend.ServiceApp.Submit.Error
	if errorKey, exists := resultMap["Backend.ServiceApp.Submit.Error"]; exists {
		expectedTranslations := map[string]string{
			"en": "Failed to submit",
			"fr": "Échec de la soumission",
			"de": "Übermittlung fehlgeschlagen",
		}
		checkTranslations(t, "Backend.ServiceApp.Submit.Error", errorKey.Translations, expectedTranslations)
	} else {
		t.Error("Key 'Backend.ServiceApp.Submit.Error' not found in result")
	}

	// Проверяем ключ Backend.ServiceApp.Submit.Success
	if successKey, exists := resultMap["Backend.ServiceApp.Submit.Success"]; exists {
		expectedTranslations := map[string]string{
			"en": "Successfully submitted",
			"fr": "Soumis avec succès",
			"de": "Erfolgreich übermittelt",
		}
		checkTranslations(t, "Backend.ServiceApp.Submit.Success", successKey.Translations, expectedTranslations)
	} else {
		t.Error("Key 'Backend.ServiceApp.Submit.Success' not found in result")
	}

	// Проверяем ключ Backend.ServiceApp.HelloFriend
	if helloKey, exists := resultMap["Backend.ServiceApp.HelloFriend"]; exists {
		expectedTranslations := map[string]string{
			"en": "Hello {name}!",
			"fr": "Bonjorn {name}!",
		}
		checkTranslations(t, "Backend.ServiceApp.HelloFriend", helloKey.Translations, expectedTranslations)
	} else {
		t.Error("Key 'Backend.ServiceApp.HelloFriend' not found in result")
	}
}

// Вспомогательная функция для проверки переводов
func checkTranslations(t *testing.T, keyName string, actual, expected map[string]string) {
	if len(actual) != len(expected) {
		t.Errorf("Key %s: expected %d translations, got %d", keyName, len(expected), len(actual))
	}

	for lang, expectedText := range expected {
		if actualText, exists := actual[lang]; !exists {
			t.Errorf("Key %s: translation for language '%s' not found", keyName, lang)
		} else if actualText != expectedText {
			t.Errorf("Key %s.%s: expected '%s', got '%s'", keyName, lang, expectedText, actualText)
		}
	}
}

// Тест с пустыми данными
func TestConvertDictionaryToImportKeys_Empty(t *testing.T) {
	input := translations.Dictionary{}
	result := convertDictionaryToImportKeys(input, []string{})

	if len(result) != 0 {
		t.Errorf("Expected empty result for empty input, got %d items", len(result))
	}
}

// Тест с частичными переводами (не все ключи имеют все языки)
func TestConvertDictionaryToImportKeys_PartialTranslations(t *testing.T) {
	input := translations.Dictionary{
		"en": {
			"key1": "text1 en",
			"key2": "text2 en",
		},
		"fr": {
			"key1": "text1 fr",
			// key2 отсутствует во французском
		},
	}

	result := convertDictionaryToImportKeys(input, []string{})

	if len(result) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(result))
	}

	// Находим key2 и проверяем, что у него только английский перевод
	for _, key := range result {
		if key.Name == "key2" {
			if len(key.Translations) != 1 {
				t.Errorf("Key 'key2' should have 1 translation, got %d", len(key.Translations))
			}
			if key.Translations["en"] != "text2 en" {
				t.Errorf("Key 'key2'.en: expected 'text2 en', got '%s'", key.Translations["en"])
			}
			break
		}
	}
}
