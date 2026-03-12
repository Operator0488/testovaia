package s3client

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

func (m *minioClient) Delete(ctx context.Context, key string) error {
	if key == "" {
		return ErrKeyRequired
	}

	err := m.client.RemoveObject(ctx, m.config.Get().Bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}
