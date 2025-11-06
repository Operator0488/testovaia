package config

import (
	"errors"
	"os"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/vault"
	"github.com/hashicorp/consul/api"
)

const (
	defaultVaultAddr          = "http://vault:8200"
	defaultConsulAddr         = "consul:8500"
	defaultVaultMount         = "kv"
	defaultConsulSharedPrefix = "shared"
	defaultVaultSharedPrefix  = "shared"
)

const (
	envVaultAddr     = "VAULT_ADDR"
	envVaultMount    = "VAULT_MOUNT_PATH"
	envVaultToken    = "VAULT_TOKEN_PATH"
	EnvVaultDisabled = "VAULT_DISABLED"

	envConsulAddress   = "CONSUL_ADDR"
	envConsulToken     = "CONSUL_TOKEN"
	envConsulTokenPath = "CONSUL_TOKEN_PATH"
	EnvConsulDisabled  = "CONSUL_DISABLED"

	EnvAppName            = "app.name"
	envConsulAppPrefix    = "consul.app_prefix"
	envConsulSharedPrefix = "consul.shared_prefix"
	envVaultAppPath       = "vault.app_path"
	envVaultSharedPath    = "vault.shared_path"
)

func getVaultMount() string {
	str, _ := os.LookupEnv(envVaultMount)
	if len(str) == 0 {
		return defaultVaultMount
	}

	return str
}

func getVaultConfig() (*vault.VaultConfig, error) {
	disabled, ok := os.LookupEnv(EnvVaultDisabled)
	if ok && disabled == "true" {
		return nil, nil
	}

	addr, ok := os.LookupEnv(envVaultAddr)
	if !ok {
		addr = defaultVaultAddr
	}

	token, ok := os.LookupEnv(envVaultToken)
	if !ok {
		return nil, errors.New("vault token is required")
	}

	return &vault.VaultConfig{
		Address: addr,
		Token:   token,
	}, nil
}

func getConsulConfig() (*api.Config, error) {
	disabled, ok := os.LookupEnv(EnvConsulDisabled)
	if ok && disabled == "true" {
		return nil, nil
	}
	addr, ok := os.LookupEnv(envConsulAddress)
	if !ok {
		addr = defaultConsulAddr
	}
	token, _ := os.LookupEnv(envConsulToken)
	tokenFile, _ := os.LookupEnv(envConsulTokenPath)
	return &api.Config{
		Address:   addr,
		Token:     token,
		TokenFile: tokenFile,
	}, nil
}
