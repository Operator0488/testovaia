package s3client

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

// Download file from storage
func (m *minioClient) Download(ctx context.Context, key string) (*DownloadOutput, error) {
	if key == "" {
		return nil, ErrKeyRequired
	}

	object, err := m.client.GetObject(ctx, m.config.Get().Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	objInfo, err := object.Stat()
	if err != nil {
		object.Close()
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &DownloadOutput{
		Body:        object,
		ContentType: objInfo.ContentType,
		Size:        objInfo.Size,
		Metadata:    objInfo.UserMetadata,
	}, nil
}
