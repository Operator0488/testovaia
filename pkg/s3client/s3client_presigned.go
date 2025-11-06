package s3client

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// PresignedGetObject create temp signed URL for open file
func (m *minioClient) PresignedGetObject(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if key == "" {
		return "", ErrKeyRequired
	}

	if ttl <= 0 {
		return "", errors.New("ttl must be positive")
	}

	url, err := m.client.PresignedGetObject(ctx, m.config.Get().Bucket, key, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}
