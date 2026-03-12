package s3client

import (
	"context"
	"errors"
	"fmt"

	"github.com/minio/minio-go/v7"
)

// Upload file to storage
func (m *minioClient) Upload(ctx context.Context, input *UploadInput) error {
	if input == nil {
		return errors.New("upload input cannot be nil")
	}

	if input.Key == "" {
		return ErrKeyRequired
	}

	if input.Body == nil {
		return errors.New("object body is required")
	}

	opts := minio.PutObjectOptions{}

	if input.ContentType != "" {
		opts.ContentType = input.ContentType
	}

	if input.Metadata != nil {
		opts.UserMetadata = input.Metadata
	}

	_, err := m.client.PutObject(ctx, m.config.Get().Bucket, input.Key, input.Body, -1, opts)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	return nil
}
