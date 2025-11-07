package storage

import (
	"context"
	"io"
	"mime/multipart"
	"time"
)

// Metadata contains information about an uploaded/stored file
type Metadata struct {
	Key         string            // Storage key/path
	FileName    string            // Original filename
	ContentType string            // MIME type
	URL         string            // Access URL
	Size        int64             // File size in bytes
	ETag        string            // Entity tag for caching
	Width       int               // Image width (0 if not applicable)
	Height      int               // Image height (0 if not applicable)
	ModTime     time.Time         // Last modified time
	Custom      map[string]string // Custom metadata
}

// UploadOptions configures file upload behavior
type UploadOptions struct {
	// Key is the storage path/key. If empty, auto-generated from filename
	Key string

	// ContentType overrides auto-detection
	ContentType string

	// Metadata allows custom key-value pairs
	Metadata map[string]string

	// ExtractImageDimensions automatically extracts width/height for images
	ExtractImageDimensions bool

	// CacheControl sets cache control headers
	CacheControl string

	// Public makes the file publicly accessible (if supported)
	Public bool
}

// DownloadOptions configures file download behavior
type DownloadOptions struct {
	// Range specifies byte range (e.g., "bytes=0-1023")
	Range string

	// IfModifiedSince for conditional downloads
	IfModifiedSince *time.Time
}

// DeleteOptions configures file deletion behavior
type DeleteOptions struct {
	// Recursive deletes all files with matching prefix
	Recursive bool
}

// ListOptions configures file listing behavior
type ListOptions struct {
	// Prefix filters by key prefix
	Prefix string

	// MaxResults limits number of results
	MaxResults int

	// ContinuationToken for pagination
	ContinuationToken string
}

// ListResult contains list operation results
type ListResult struct {
	Items             []*Metadata
	ContinuationToken string
	HasMore           bool
}

// Storage is the unified interface for all storage operations
type Storage interface {
	// Upload uploads a file from an io.Reader
	Upload(ctx context.Context, reader io.Reader, opts *UploadOptions) (*Metadata, error)

	// UploadMultipart uploads from a multipart form file (convenience method)
	UploadMultipart(ctx context.Context, fileHeader *multipart.FileHeader, opts *UploadOptions) (*Metadata, error)

	// Download retrieves a file
	Download(ctx context.Context, key string, opts *DownloadOptions) (io.ReadCloser, error)

	// Delete removes a file
	Delete(ctx context.Context, key string, opts *DeleteOptions) error

	// Exists checks if a file exists
	Exists(ctx context.Context, key string) (bool, error)

	// GetMetadata retrieves file metadata without downloading
	GetMetadata(ctx context.Context, key string) (*Metadata, error)

	// List lists files matching criteria
	List(ctx context.Context, opts *ListOptions) (*ListResult, error)

	// GetURL returns a public URL for accessing the file
	GetURL(ctx context.Context, key string) (string, error)

	// GetSignedURL returns a temporary signed URL (if supported)
	GetSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// Helper function to create default upload options
func DefaultUploadOptions(key string) *UploadOptions {
	return &UploadOptions{
		Key:                    key,
		ExtractImageDimensions: true,
		Public:                 false,
		Metadata:               make(map[string]string),
	}
}

// Helper function for download options
func DefaultDownloadOptions() *DownloadOptions {
	return &DownloadOptions{}
}

// Helper function for delete options
func DefaultDeleteOptions() *DeleteOptions {
	return &DeleteOptions{
		Recursive: false,
	}
}

// Helper function for list options
func DefaultListOptions() *ListOptions {
	return &ListOptions{
		MaxResults: 1000,
	}
}
