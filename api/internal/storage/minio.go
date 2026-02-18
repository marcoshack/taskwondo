package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage implements Storage using MinIO/S3.
type MinIOStorage struct {
	client *minio.Client
	bucket string
}

// NewMinIOStorage creates a new MinIOStorage.
func NewMinIOStorage(endpoint, accessKey, secretKey, bucket, region string, useSSL bool) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("creating minio client: %w", err)
	}
	return &MinIOStorage{client: client, bucket: bucket}, nil
}

// Put stores a file in MinIO.
func (s *MinIOStorage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*ObjectInfo, error) {
	info, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("putting object %q: %w", key, err)
	}
	return &ObjectInfo{Key: key, Size: info.Size, ContentType: contentType}, nil
}

// Get retrieves a file from MinIO.
func (s *MinIOStorage) Get(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("getting object %q: %w", key, err)
	}
	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, nil, fmt.Errorf("stat object %q: %w", key, err)
	}
	return obj, &ObjectInfo{Key: key, Size: stat.Size, ContentType: stat.ContentType}, nil
}

// Delete removes a file from MinIO.
func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("deleting object %q: %w", key, err)
	}
	return nil
}
