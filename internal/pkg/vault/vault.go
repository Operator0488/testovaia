package vault

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/api"
)

type VaultClient struct {
	client *api.Client
}

type VaultConfig struct {
	Address string
	Token   string
}

func NewClient(cfg VaultConfig) (*VaultClient, error) {
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = cfg.Address

	client, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(cfg.Token)

	return &VaultClient{
		client: client,
	}, nil
}

func (vl *VaultClient) LoadKV(ctx context.Context, mount, path string) (map[string]interface{}, error) {
	v, err := vl.client.KVv2(mount).Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read KV secrets for path:'%s' and mount: '%s', %w", path, mount, err)
	}
	if v.Data == nil {
		return nil, fmt.Errorf("empty KV secrets for %s", path)
	}
	return v.Data, nil
}

func (vl *VaultClient) LoadPKI(ctx context.Context, path string) (interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf("pki path is required for read-only mode")
	}

	secret, err := vl.client.Logical().Read(path)

	//secret, err := vl.client.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read PKI secret: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("empty response from PKI at path %s", path)
	}

	value, exists := secret.Data["certificate"] // TODO какие поля будут нужны
	if !exists {
		return nil, fmt.Errorf("key certificate not found in PKI secret at %s", path)
	}

	return value, nil
}
