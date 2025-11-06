package s3client

import (
	"context"
	"errors"
	"fmt"

	"github.com/minio/minio-go/v7"
)

// Exist file in storage
func (m *minioClient) Exist(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, ErrKeyRequired
	}

	_, err := m.client.StatObject(ctx, m.config.Get().Bucket, key, minio.StatObjectOptions{})
	if err != nil {
		var minioErr minio.ErrorResponse
		if errors.As(err, &minioErr) && minioErr.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}
