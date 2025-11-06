// Package config for get env, vault config values
package config

import (
	"context"
	"fmt"
	"sync"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/config"
	configprovider "git.vepay.dev/knoknok/backend-platform/internal/pkg/config_provider"
	consulprovider "git.vepay.dev/knoknok/backend-platform/internal/pkg/config_provider/consul"
	vaultprovider "git.vepay.dev/knoknok/backend-platform/internal/pkg/config_provider/vault"
	"git.vepay.dev/knoknok/backend-platform/internal/pkg/consul"
	"git.vepay.dev/knoknok/backend-platform/internal/pkg/vault"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

var (
	configInstance *config.Config
	once           sync.Once
)

const (
	defaultFileName = "config"
)

func GetConfig() config.Configurer {
	if configInstance == nil {
		logger.Fatal(context.Background(), "Config not initialized. Call Init first")
	}
	return configInstance
}

func Init(ctx context.Context, opts ...InitOption) error {
	var err error
	once.Do(func() {
		o := applyInitOptions(opts...)

		cfg := config.New(o.ConfigPath, o.FileName)

		if err = cfg.LoadEnv(ctx); err != nil {
			logger.Fatal(ctx, "failed init config", logger.Err(err))
		}

		appName, err := getAppName(cfg)
		if err != nil {
			logger.Fatal(ctx, "failed init config", logger.Err(err))
		}

		// apply consul configs
		consulClient, err := getConsulClient()
		if err != nil {
			logger.Fatal(ctx, "failed init config", logger.Err(err))
		}

		if consulClient != nil {
			sharedPrefix := cfg.GetStringOrDefault(envConsulSharedPrefix, defaultConsulSharedPrefix)
			appPrefix := cfg.GetStringOrDefault(envConsulAppPrefix, appName)
			consulShared := consulprovider.NewProvider(sharedPrefix, consulClient)
			consulApp := consulprovider.NewProvider(appPrefix, consulClient)

			mustLoadProvider(ctx, cfg, consulShared)
			mustLoadProvider(ctx, cfg, consulApp)

			if err := cfg.Bootstrap(ctx, consulApp); err != nil {
				logger.Fatal(ctx, "failed boostrap config", logger.Err(err))
			}
		}

		// apply vault configs
		vaultClient, err := getVaultClient()
		if err != nil {
			logger.Fatal(ctx, "failed init config", logger.Err(err))
		}

		if vaultClient != nil {
			mount := getVaultMount()
			appPath := cfg.GetStringOrDefault(envVaultAppPath, appName)
			sharedPath := cfg.GetStringOrDefault(envVaultSharedPath, defaultVaultSharedPrefix)

			sharedProvider := vaultprovider.NewProvider(sharedPath, mount, vaultClient)
			appProvider := vaultprovider.NewProvider(appPath, mount, vaultClient)

			mustLoadProvider(ctx, cfg, sharedProvider)
			mustLoadProvider(ctx, cfg, appProvider)
		}

		configInstance = cfg
	})
	return err
}

func getVaultClient() (*vault.VaultClient, error) {
	vaultConfig, err := getVaultConfig()
	if err != nil || vaultConfig == nil {
		return nil, err
	}

	client, err := vault.NewClient(*vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client %w", err)
	}

	return client, nil
}

func getConsulClient() (consul.Client, error) {
	consulConfig, err := getConsulConfig()
	if err != nil || consulConfig == nil {
		return nil, err
	}

	client, err := consul.NewClientWithConfig(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client %w", err)
	}
	return client, nil
}

func getAppName(cfg *config.Config) (string, error) {
	appName := cfg.GetString(EnvAppName)
	if len(appName) == 0 {
		return "", fmt.Errorf("required parameter %s is not set", EnvAppName)
	}
	return appName, nil
}

func mustLoadProvider(ctx context.Context, cfg *config.Config, provider configprovider.Provider) {
	if provider == nil {
		return
	}
	if err := cfg.LoadFromProvider(ctx, provider); err != nil {
		logger.Fatal(ctx, "failed init config", logger.Err(err))
	}
}
