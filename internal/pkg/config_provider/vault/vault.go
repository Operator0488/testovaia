package vault

import (
	"context"

	configprovider "easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/config_provider"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/vault"
)

type vaultProvider struct {
	path   string
	mount  string
	client *vault.VaultClient
}

func NewProvider(path, mount string, client *vault.VaultClient) configprovider.Provider {
	return &vaultProvider{
		client: client,
		path:   path,
		mount:  mount,
	}
}

// Close implements configprovider.Provider.
func (c *vaultProvider) Close(ctx context.Context) error {
	return nil
}

// Get implements configprovider.Provider.
func (c *vaultProvider) Get(ctx context.Context) (configprovider.ConfigData, error) {
	return c.client.LoadKV(ctx, c.mount, c.path)
}

// Set implements configprovider.Provider.
func (c *vaultProvider) Set(ctx context.Context, value configprovider.ConfigData) error {
	return configprovider.ErrUnsupported
}

// Watch implements configprovider.Provider.
func (c *vaultProvider) Watch(ctx context.Context, onChange func(map[string]interface{})) error {
	return configprovider.ErrUnsupported
}
