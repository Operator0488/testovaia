package s3client

import (
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// rebuildClient create new s3 client, start new HealthCheck, update config.
func (m *minioClient) rebuildClient(config *Config) error {
	mClient, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
		Region: config.Region,
	})
	if err != nil {
		return fmt.Errorf("failed to create minio client: %w", err)
	}

	if err := m.checkBucket(mClient, config); err != nil {
		return err
	}

	cancelHealthCheck, err := mClient.HealthCheck(defaultHealthCheckDuration)
	if err != nil {
		return fmt.Errorf("failed to start health check: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancelHealthCheck != nil {
		m.cancelHealthCheck()
	}

	m.cancelHealthCheck = cancelHealthCheck
	m.client = mClient
	return nil
}
