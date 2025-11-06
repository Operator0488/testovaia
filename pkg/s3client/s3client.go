package s3client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	defaultTimeout             = 30 * time.Second
	defaultRegion              = "us-east-1"
	defaultHealthCheckDuration = 10 * time.Second // how many time client wait next check connection operation
	defaultPingRetry           = 8                // how many time pinger check client status
	defaultPingDuration        = 3 * time.Second  // how many time pinger wait between each try
)

var (
	ErrSameKey         = errors.New("source and destination keys cannot be the same")
	ErrKeyRequired     = errors.New("object key is required")
	ErrClientIsOffline = errors.New("client is offline")
)

type UploadInput struct {
	Key         string            `json:"key"`
	Body        io.Reader         `json:"-"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata"`
}

type DownloadOutput struct {
	Body        io.ReadCloser     `json:"-"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Metadata    map[string]string `json:"metadata"`
}

// Client S3
type Client interface {
	// Upload file to storage
	Upload(ctx context.Context, input *UploadInput) error

	// Download file from storage
	Download(ctx context.Context, key string) (*DownloadOutput, error)

	// Exist file in storage
	Exist(ctx context.Context, key string) (bool, error)

	// Delete file from storage
	Delete(ctx context.Context, key string) error

	// Move file in storage or rename
	Move(ctx context.Context, sourceKey, destinationKey string) error

	// PresignedGetObject create temp signed URL for open file
	PresignedGetObject(ctx context.Context, key string, ttl time.Duration) (string, error)

	// Ping check s3 connection
	Ping(ctx context.Context) error

	// Close for close connection in gracefull shutdown flow
	Close() error
}

type minioClient struct {
	client            *minio.Client
	config            config.IConfigWatcher[*Config]
	cancelHealthCheck context.CancelFunc
	mu                sync.Mutex
}

// NewClient creating new client for S3
func NewClient(config config.IConfigWatcher[*Config]) (Client, error) {
	if err := config.Get().Validate(); err != nil {
		return nil, err
	}

	mClient, err := minio.New(config.Get().Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.Get().AccessKeyID, config.Get().SecretAccessKey, ""),
		Secure: config.Get().UseSSL,
		Region: config.Get().Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	client := &minioClient{
		client: mClient,
		config: config,
	}

	if err := client.checkBucket(mClient, config.Get()); err != nil {
		return nil, err
	}

	cancelHealthCheck, err := mClient.HealthCheck(defaultHealthCheckDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to start health check: %w", err)
	}

	client.cancelHealthCheck = cancelHealthCheck
	config.OnRefresh(client.rebuildClient)

	return client, nil
}

// Close
func (m *minioClient) Close() error {
	if m.cancelHealthCheck != nil {
		m.cancelHealthCheck()
	}
	return nil
}

// checkBucket check bucket is exist, if does not exist trying create bucket if create_bicket flag is true
func (m *minioClient) checkBucket(mClient *minio.Client, config *Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	exists, err := mClient.BucketExists(ctx, config.Bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if !config.CreateBucket {
			return fmt.Errorf("bucket %s does not exist", config.Bucket)
		}

		err := mClient.MakeBucket(ctx, config.Bucket, minio.MakeBucketOptions{
			Region: config.Region,
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

func (m *minioClient) Ping(ctx context.Context) error {
	if m.client.IsOnline() {
		return nil
	}
	i := defaultPingRetry
	ticker := time.NewTicker(defaultPingDuration)
	defer ticker.Stop()

	for i > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if m.client.IsOnline() {
				return nil
			}
			i--
		}
	}
	return ErrClientIsOffline
}
