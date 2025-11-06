package tolgee

import (
	"context"
	"errors"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/config"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

const (
	defaultHost = "tolgee:8089"
)

var (
	ErrDisabled = errors.New("tolgee disabled")
)

type Config struct {
	Host      string
	APIKey    string
	ProjectID string
	Enabled   bool
	Tags      []string
}

func NewConfig(cfg config.Configurer) *Config {
	return &Config{
		Enabled:   cfg.GetBool("tolgee.enabled"),
		Host:      cfg.GetStringOrDefault("tolgee.host", defaultHost),
		APIKey:    cfg.GetString("tolgee.api_key"),
		ProjectID: cfg.GetString("tolgee.project_id"),
		Tags:      getTags(cfg),
	}
}

// getTags добавляет теги backend и {app.name} как обязательные, если их нет.
func getTags(cfg config.Configurer) []string {
	tags := cfg.GetStringSlice("tolgee.tags")
	logger.Info(context.Background(), "TAAAAGS", logger.Any("tags", tags))
	appTag := cfg.GetString("app.name")
	backendTag := "backend"

	appExist, backendExist := false, false

	for _, tag := range tags {
		if tag == appTag {
			appExist = true
		}
		if tag == backendTag {
			backendExist = true
		}
	}

	if !appExist {
		tags = append(tags, appTag)
	}

	if !backendExist {
		tags = append(tags, backendTag)
	}

	logger.Info(context.Background(), "FINAL TAAAAGS",
		logger.Any("tags", tags),
		logger.Bool("appExist", appExist),
		logger.Bool("backendExist", backendExist),
	)

	return tags
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return ErrDisabled
	}

	if len(c.Host) == 0 {
		return errors.New("tolgee tolgee.host is required")
	}

	if len(c.APIKey) == 0 {
		return errors.New("tolgee tolgee.api_key is required")
	}

	if len(c.ProjectID) == 0 {
		return errors.New("tolgee tolgee.project_id is required")
	}
	return nil
}
