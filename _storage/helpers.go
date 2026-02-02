package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// UploadFile is a convenience function to upload a file with minimal configuration
func UploadFile(ctx context.Context, s Storage, key string, reader io.Reader, contentType string) (*Metadata, error) {
	opts := &UploadOptions{
		Key:                    key,
		ContentType:            contentType,
		ExtractImageDimensions: true,
		Metadata:               make(map[string]string),
	}
	return s.Upload(ctx, reader, opts)
}

// UploadBytes uploads byte data as a file
func UploadBytes(ctx context.Context, s Storage, key string, data []byte, contentType string) (*Metadata, error) {
	return UploadFile(ctx, s, key, bytes.NewReader(data), contentType)
}

// UploadMultipartSimple uploads a multipart file with a simple key
func UploadMultipartSimple(ctx context.Context, s Storage, key string, fileHeader *multipart.FileHeader) (*Metadata, error) {
	opts := &UploadOptions{
		Key:                    key,
		ExtractImageDimensions: true,
		Metadata:               make(map[string]string),
	}
	return s.UploadMultipart(ctx, fileHeader, opts)
}

// UploadWithMetadata uploads a file with custom metadata
func UploadWithMetadata(ctx context.Context, s Storage, key string, reader io.Reader, contentType string, metadata map[string]string) (*Metadata, error) {
	opts := &UploadOptions{
		Key:                    key,
		ContentType:            contentType,
		ExtractImageDimensions: true,
		Metadata:               metadata,
	}
	return s.Upload(ctx, reader, opts)
}

// UploadPublic uploads a file and makes it publicly accessible
func UploadPublic(ctx context.Context, s Storage, key string, reader io.Reader, contentType string) (*Metadata, error) {
	opts := &UploadOptions{
		Key:                    key,
		ContentType:            contentType,
		ExtractImageDimensions: true,
		Public:                 true,
		Metadata:               make(map[string]string),
	}
	return s.Upload(ctx, reader, opts)
}

// UploadWithCache uploads a file with cache control headers
func UploadWithCache(ctx context.Context, s Storage, key string, reader io.Reader, contentType string, cacheControl string) (*Metadata, error) {
	opts := &UploadOptions{
		Key:                    key,
		ContentType:            contentType,
		ExtractImageDimensions: true,
		CacheControl:           cacheControl,
		Metadata:               make(map[string]string),
	}
	return s.Upload(ctx, reader, opts)
}

// DownloadToBytes downloads a file and returns it as bytes
func DownloadToBytes(ctx context.Context, s Storage, key string) ([]byte, error) {
	reader, err := s.Download(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// DeleteRecursive deletes all files with the given prefix
func DeleteRecursive(ctx context.Context, s Storage, prefix string) error {
	return s.Delete(ctx, prefix, &DeleteOptions{Recursive: true})
}

// DeleteSimple deletes a single file
func DeleteSimple(ctx context.Context, s Storage, key string) error {
	return s.Delete(ctx, key, nil)
}

// CopyFile copies a file from one key to another (requires download + upload)
func CopyFile(ctx context.Context, s Storage, srcKey, destKey string) (*Metadata, error) {
	// Get source metadata
	srcMeta, err := s.GetMetadata(ctx, srcKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get source metadata: %w", err)
	}

	// Download source
	reader, err := s.Download(ctx, srcKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download source: %w", err)
	}
	defer reader.Close()

	// Upload to destination
	opts := &UploadOptions{
		Key:          destKey,
		ContentType:  srcMeta.ContentType,
		Metadata:     srcMeta.Custom,
		CacheControl: "",
	}

	return s.Upload(ctx, reader, opts)
}

// ListAll lists all files in storage (be careful with large buckets)
func ListAll(ctx context.Context, s Storage) ([]*Metadata, error) {
	result, err := s.List(ctx, &ListOptions{
		MaxResults: 0, // unlimited
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// ListByPrefix lists all files with a given prefix
func ListByPrefix(ctx context.Context, s Storage, prefix string) ([]*Metadata, error) {
	result, err := s.List(ctx, &ListOptions{
		Prefix:     prefix,
		MaxResults: 0,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// GetTemporaryURL generates a temporary signed URL valid for the specified duration
func GetTemporaryURL(ctx context.Context, s Storage, key string, validFor time.Duration) (string, error) {
	return s.GetSignedURL(ctx, key, validFor)
}

// GetPublicURL returns a permanent public URL for a file
func GetPublicURL(ctx context.Context, s Storage, key string) (string, error) {
	return s.GetURL(ctx, key)
}

// FileExists checks if a file exists in storage
func FileExists(ctx context.Context, s Storage, key string) (bool, error) {
	return s.Exists(ctx, key)
}

// GetFileInfo retrieves file information without downloading
func GetFileInfo(ctx context.Context, s Storage, key string) (*Metadata, error) {
	return s.GetMetadata(ctx, key)
}

// GenerateKey generates a unique storage key from a filename
func GenerateKey(prefix, filename string) string {
	timestamp := time.Now().Unix()
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]
	return fmt.Sprintf("%s/%s_%d%s", prefix, base, timestamp, ext)
}

// GenerateUniqueKey generates a unique key with UUID-like identifier
func GenerateUniqueKey(prefix, filename string) string {
	timestamp := time.Now().UnixNano()
	ext := filepath.Ext(filename)
	return fmt.Sprintf("%s/%d_%s%s", prefix, timestamp, "file", ext)
}

// StreamToFiber streams a file directly to a Fiber response with proper headers
func StreamToFiber(c *fiber.Ctx, s Storage, key string) error {
	ctx := c.Context()

	// Get metadata first
	meta, err := s.GetMetadata(ctx, key)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	// Download the file
	reader, err := s.Download(ctx, key, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to download file",
		})
	}
	defer reader.Close()

	// Set headers
	c.Set(fiber.HeaderContentType, meta.ContentType)
	c.Set(fiber.HeaderContentLength, strconv.FormatInt(meta.Size, 10))
	c.Set(fiber.HeaderCacheControl, "public, max-age=31536000")
	c.Set(fiber.HeaderETag, meta.ETag)

	// Stream the file
	return c.SendStream(reader)
}

// ProxyToFiber proxies a file with support for range requests and caching
func ProxyToFiber(c *fiber.Ctx, s Storage, key string, opts *ProxyOptions) error {
	if opts == nil {
		opts = DefaultProxyOptions()
	}

	ctx := c.Context()

	// Get metadata first
	meta, err := s.GetMetadata(ctx, key)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	// Check ETag for caching
	if opts.EnableCaching {
		ifNoneMatch := c.Get(fiber.HeaderIfNoneMatch)
		if ifNoneMatch == meta.ETag {
			return c.SendStatus(fiber.StatusNotModified)
		}
	}

	// Handle range requests
	rangeHeader := c.Get(fiber.HeaderRange)
	var downloadOpts *DownloadOptions
	if rangeHeader != "" && opts.EnableRangeRequests {
		downloadOpts = &DownloadOptions{
			Range: rangeHeader,
		}
	} else {
		downloadOpts = DefaultDownloadOptions()
	}

	// Download the file
	reader, err := s.Download(ctx, key, downloadOpts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to download file",
		})
	}
	defer reader.Close()

	// Set headers
	c.Set(fiber.HeaderContentType, meta.ContentType)
	c.Set(fiber.HeaderETag, meta.ETag)
	c.Set(fiber.HeaderLastModified, meta.ModTime.Format(time.RFC1123))
	c.Set(fiber.HeaderAcceptRanges, "bytes")

	if opts.CacheControl != "" {
		c.Set(fiber.HeaderCacheControl, opts.CacheControl)
	} else {
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000")
	}

	if opts.Attachment {
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", meta.FileName))
	} else {
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("inline; filename=%q", meta.FileName))
	}

	// Set content length for non-range requests
	if rangeHeader == "" {
		c.Set(fiber.HeaderContentLength, strconv.FormatInt(meta.Size, 10))
		return c.SendStream(reader)
	}

	// For range requests, let Fiber handle it
	c.Status(fiber.StatusPartialContent)
	return c.SendStream(reader)
}

// ProxyOptions configures file proxying behavior
type ProxyOptions struct {
	// EnableCaching enables ETag-based caching
	EnableCaching bool

	// EnableRangeRequests enables HTTP range request support
	EnableRangeRequests bool

	// CacheControl sets the Cache-Control header
	CacheControl string

	// Attachment forces download instead of inline display
	Attachment bool
}

// DefaultProxyOptions returns sensible defaults for proxying
func DefaultProxyOptions() *ProxyOptions {
	return &ProxyOptions{
		EnableCaching:       true,
		EnableRangeRequests: true,
		CacheControl:        "public, max-age=31536000",
		Attachment:          false,
	}
}

// ServeFile is a simple helper to serve a file through Fiber
func ServeFile(c *fiber.Ctx, s Storage, key string) error {
	return StreamToFiber(c, s, key)
}

// DownloadFile forces a file download through Fiber
func DownloadFile(c *fiber.Ctx, s Storage, key string) error {
	opts := DefaultProxyOptions()
	opts.Attachment = true
	return ProxyToFiber(c, s, key, opts)
}
