package checkins

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config holds the minimal configuration needed to connect to S3 or any
// S3-compatible store (MinIO, Backblaze, etc.).
type S3Config struct {
	Endpoint        string // leave empty for AWS; set to MinIO URL for local dev
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool // required for MinIO
}

type S3Storage struct {
	client *s3.Client
	bucket string
}

func NewS3Storage(cfg S3Config) *S3Storage {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, opts ...any) (aws.Endpoint, error) {
		if cfg.Endpoint != "" {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: cfg.UsePathStyle,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg := aws.Config{
		Region:                      cfg.Region,
		Credentials:                 credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		EndpointResolverWithOptions: resolver,
	}

	opts := []func(*s3.Options){
		func(o *s3.Options) {
			if cfg.UsePathStyle {
				o.UsePathStyle = true
			}
		},
	}

	return &S3Storage{
		client: s3.NewFromConfig(awsCfg, opts...),
		bucket: cfg.Bucket,
	}
}

func (s *S3Storage) Put(ctx context.Context, key string, r io.Reader, size int64, mimeType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          r,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(mimeType),
	})
	if err != nil {
		return fmt.Errorf("s3 put %s: %w", key, err)
	}
	return nil
}

func (s *S3Storage) ObjectKey(checkInID int64, ext string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("evidence/%d/%s%s", checkInID, hex.EncodeToString(b), ext)
}

// NoopStorage is used when S3 is not configured (e.g. test / no-file mode).
// File evidence is rejected gracefully at the handler level when storage is nil,
// so this implementation panics to make misuse obvious during development.
type NoopStorage struct{}

func (n NoopStorage) Put(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return fmt.Errorf("object storage is not configured")
}

func (n NoopStorage) ObjectKey(checkInID int64, ext string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("evidence/%d/%s%s", checkInID, hex.EncodeToString(b), ext)
}
