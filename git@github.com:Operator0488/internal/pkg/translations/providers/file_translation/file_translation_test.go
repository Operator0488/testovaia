package filetranslation

import (
	"context"
	"io/fs"
	"os"
	"testing"
)

func getFS() fs.FS {
	return os.DirFS(".")
}

func TestNewProvider(t *testing.T) {
	fsys := getFS()
	provider := NewProvider(fsys, "test_translations.json")

	if provider == nil {
		t.Fatal("NewProvider returned nil")
	}

	// Проверяем, что провайдер реализует интерфейс
	var _ FileProvider = provider
}

// TestGetTranslations_ExpandedStructure тестирует загрузку переводов в развернутой структуре по языкам
func TestGetTranslations_ExpandedStructure(t *testing.T) {
	fsys := getFS()
	provider := NewProvider(fsys, "test_translations.json")
	ctx := context.Background()

	dict, err := provider.GetTranslations(ctx, nil)
	if err != nil {
		t.Fatalf("GetTranslations failed: %v", err)
	}

	// Ожидаемая структура данных по языкам
	expectedStructure := map[string]map[string]string{
		"en": {
			"Backend.ServiceApp.Submit.Error":   "Failed to submit",
			"Backend.ServiceApp.Submit.Success": "Successfully submitted",
		},
		"fr": {
			"Backend.ServiceApp.Submit.Error":   "Échec de la soumission",
			"Backend.ServiceApp.Submit.Success": "Soumis avec succès",
		},
		"de": {
			"Backend.ServiceApp.Submit.Error":   "Übermittlung fehlgeschlagen",
			"Backend.ServiceApp.Submit.Success": "Erfolgreich übermittelt",
		},
	}

	// Проверяем наличие всех языков
	if len(dict) != len(expectedStructure) {
		t.Errorf("Expected %d languages, got %d", len(expectedStructure), len(dict))
	}

	// Проверяем каждую языковую группу
	for expectedLang, expectedTranslations := range expectedStructure {
		actualTranslations, exists := dict[expectedLang]
		if !exists {
			t.Errorf("Language %s not found in result", expectedLang)
			continue
		}

		// Проверяем каждую пару ключ-значение
		for expectedKey, expectedText := range expectedTranslations {
			actualText, exists := actualTranslations[expectedKey]
			if !exists {
				t.Errorf("Key %s not found in %s translations", expectedKey, expectedLang)
				continue
			}
			if actualText != expectedText {
				t.Errorf("Translation for %s.%s: expected '%s', got '%s'",
					expectedLang, expectedKey, expectedText, actualText)
			}
		}

		// Проверяем, что нет лишних ключей
		if len(actualTranslations) != len(expectedTranslations) {
			t.Errorf("Language %s: expected %d translations, got %d",
				expectedLang, len(expectedTranslations), len(actualTranslations))
		}
	}
}

func TestGetTranslations_FileNotFound(t *testing.T) {
	fsys := getFS()
	provider := NewProvider(fsys, "non_existent_file.json")
	ctx := context.Background()

	_, err := provider.GetTranslations(ctx, nil)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestGetTranslations_InvalidJSON(t *testing.T) {
	fsys := getFS()
	provider := NewProvider(fsys, "invalid_json.json")
	ctx := context.Background()

	_, err := provider.GetTranslations(ctx, nil)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}
