package s3client

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

// Move file in storage or rename
func (m *minioClient) Move(ctx context.Context, sourceKey, destinationKey string) error {
	if sourceKey == "" || destinationKey == "" {
		return ErrKeyRequired
	}

	if sourceKey == destinationKey {
		return ErrSameKey
	}

	// Копируем объект
	src := minio.CopySrcOptions{
		Bucket: m.config.Get().Bucket,
		Object: sourceKey,
	}

	dst := minio.CopyDestOptions{
		Bucket: m.config.Get().Bucket,
		Object: destinationKey,
	}

	_, err := m.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	// Удаляем исходный объект
	err = m.Delete(ctx, sourceKey)
	if err != nil {
		// Пытаемся удалить скопированный объект в случае ошибки
		m.Delete(ctx, destinationKey)
		return fmt.Errorf("failed to delete source object after copy: %w", err)
	}

	return nil
}
