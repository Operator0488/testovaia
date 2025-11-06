package s3client

import (
	"errors"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/config"
	cngf "git.vepay.dev/knoknok/backend-platform/pkg/config"
)

const (
	envS3Endpoint        = "s3.endpoint"
	envS3AccessKeyID     = "s3.access_key_id"
	envS3SecretAccessKey = "s3.secret_access_key"
	envS3Bucket          = "s3.bucket"
	envS3Region          = "s3.region"
	envS3UseSSL          = "s3.use_ssl"
	envS3CreateBucket    = "s3.create_bucket"
)

type Config struct {
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `json:"access_key_id" yaml:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key" yaml:"secret_access_key"`
	Bucket          string `json:"bucket" yaml:"bucket"`
	Region          string `json:"region" yaml:"region"`
	UseSSL          bool   `json:"use_ssl" yaml:"use_ssl"`
	CreateBucket    bool   `json:"create_bucket" yaml:"create_bucket"`
}

func NewConfig(cfg config.Configurer) *Config {
	return &Config{
		Endpoint:        cfg.GetString(envS3Endpoint),
		AccessKeyID:     cfg.GetString(envS3AccessKeyID),
		SecretAccessKey: cfg.GetString(envS3SecretAccessKey),
		Bucket:          cfg.GetStringOrDefault(envS3Bucket, cfg.GetString(cngf.EnvAppName)),
		Region:          cfg.GetStringOrDefault(envS3Region, defaultRegion),
		UseSSL:          cfg.GetBool(envS3UseSSL),
		CreateBucket:    cfg.GetBool(envS3CreateBucket),
	}
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config cannot be nil")
	}

	if c.Bucket == "" {
		return errors.New("bucket name is required")
	}

	if c.AccessKeyID == "" {
		return errors.New("access_key_id is required")
	}

	if c.SecretAccessKey == "" {
		return errors.New("secret_access_key is required")
	}

	if c.Endpoint == "" {
		return errors.New("endpoint is required")
	}
	return nil
}
