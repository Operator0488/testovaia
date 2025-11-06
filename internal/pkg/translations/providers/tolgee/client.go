package tolgee

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/config"
)

const defaultTimeout = 60 * time.Second

// Language представляет структуру языка проекта
type Language struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Tag          string `json:"tag"`
	OriginalName string `json:"originalName"`
	FlagEmoji    string `json:"flagEmoji"`
	Base         bool   `json:"base"`
}

// LanguagesResponse представляет ответ от API с языками
type LanguagesResponse struct {
	Embedded struct {
		Languages []Language `json:"languages"`
	} `json:"_embedded"`
}

type TranslationResponse map[string]map[string]string

type ImportKey struct {
	// Description описание ключа
	Description string `json:"description,omitempty"`
	// Name идентификатор ключа
	Name string `json:"name"`

	Tags []string `json:"tags"`
	// Translations переводы на разные языки
	Translations map[string]string `json:"translations"`
}

type ImportRequest struct {
	Keys []ImportKey `json:"keys"`
}

type ImportResponse struct {
	Message string `json:"message"`
}

// Client представляет клиент для работы с Tolgee API.
type Client struct {
	config     config.IConfigWatcher[*Config]
	httpClient *http.Client
}

// NewClient создает новый клиент Tolgee.
func NewClient(config config.IConfigWatcher[*Config]) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// GetTranslations получает переводы для указанных языков.
func (c *Client) GetTranslations(ctx context.Context, languages []string) (TranslationResponse, error) {
	if len(languages) == 0 {
		return nil, fmt.Errorf("failed to fetch translations, languages required")
	}

	langs := ""
	for i, lang := range languages {
		if i > 0 {
			langs += ","
		}
		langs += lang
	}

	cfg := c.config.Get()

	u, err := url.Parse(fmt.Sprintf("%s/v2/projects/%s/translations/%s", cfg.Host, cfg.ProjectID, langs))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()

	for _, tag := range cfg.Tags {
		q.Add("filterTag", tag)
	}

	q.Set("structureDelimiter", "")

	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", cfg.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var translationResponse TranslationResponse
	if err := json.NewDecoder(resp.Body).Decode(&translationResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return translationResponse, nil
}

func (c *Client) GetTags() []string {
	return c.config.Get().Tags
}

// ImportKeys импортирует ключи с переводами.
func (c *Client) ImportKeys(ctx context.Context, keys []ImportKey) error {
	cfg := c.config.Get()
	url := fmt.Sprintf("%s/v2/projects/%s/keys/import", cfg.Host, cfg.ProjectID)

	importRequest := ImportRequest{Keys: keys}
	body, err := json.Marshal(importRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", cfg.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetLanguages получает все языки проекта
func (c *Client) GetLanguages(ctx context.Context) ([]Language, error) {
	cfg := c.config.Get()
	endpoint := fmt.Sprintf("%s/v2/projects/%s/languages", cfg.Host, cfg.ProjectID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", cfg.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response LanguagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Embedded.Languages, nil
}
