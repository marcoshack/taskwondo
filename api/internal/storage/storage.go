package storage

import (
	"context"
	"io"
)

// ObjectInfo holds metadata about a stored object.
type ObjectInfo struct {
	Key         string
	Size        int64
	ContentType string
}

// Storage defines the interface for object storage.
type Storage interface {
	// Put stores a file and returns its info.
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*ObjectInfo, error)
	// Get retrieves a file and returns a ReadCloser with metadata.
	Get(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error)
	// Delete removes a file from storage.
	Delete(ctx context.Context, key string) error
}
