package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/url"
	"path"
	"strings"
	"time"

	minio "github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

type MinioStorage struct {
	client *minio.Client
	config MinioConfig
	log    *zap.Logger
}

func NewMinioStorage(cfg MinioConfig, client *minio.Client, log *zap.Logger) Storage {
	return &MinioStorage{
		client: client,
		config: cfg,
		log:    log,
	}
}

// Upload uploads a file from an io.Reader
func (s *MinioStorage) Upload(ctx context.Context, reader io.Reader, opts *UploadOptions) (*Metadata, error) {
	if opts == nil {
		return nil, fmt.Errorf("upload options cannot be nil")
	}

	key := opts.Key
	if key == "" {
		return nil, fmt.Errorf("upload key cannot be empty")
	}

	// Prepare upload options
	putOpts := minio.PutObjectOptions{
		ContentType:  opts.ContentType,
		UserMetadata: opts.Metadata,
	}

	if opts.CacheControl != "" {
		putOpts.CacheControl = opts.CacheControl
	}

	// Handle image dimension extraction if requested
	var width, height int
	if opts.ExtractImageDimensions {
		// Buffer the reader to extract dimensions
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, reader); err != nil {
			s.log.Warn("Failed to buffer for dimension extraction", zap.Error(err))
		} else {
			width, height, _ = getImageDimensions(bytes.NewReader(buf.Bytes()))
			reader = bytes.NewReader(buf.Bytes())
		}
	}

	// Detect size if possible
	var size int64 = -1
	if seeker, ok := reader.(io.Seeker); ok {
		if current, err := seeker.Seek(0, io.SeekCurrent); err == nil {
			if end, err := seeker.Seek(0, io.SeekEnd); err == nil {
				size = end
				seeker.Seek(current, io.SeekStart)
			}
		}
	}

	info, err := s.client.PutObject(ctx, s.config.BucketName, key, reader, size, putOpts)
	if err != nil {
		s.log.Error("Failed to upload", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("failed to upload %s: %w", key, err)
	}

	fileURL, err := s.buildFileURL(key)
	if err != nil {
		return nil, err
	}

	s.log.Info("Upload successful", zap.String("key", key), zap.Int64("size", info.Size))

	return &Metadata{
		Key:         key,
		FileName:    path.Base(key),
		ContentType: opts.ContentType,
		URL:         fileURL,
		Size:        info.Size,
		ETag:        info.ETag,
		Width:       width,
		Height:      height,
		ModTime:     time.Now(),
		Custom:      opts.Metadata,
	}, nil
}

// UploadMultipart uploads from a multipart form file
func (s *MinioStorage) UploadMultipart(ctx context.Context, fileHeader *multipart.FileHeader, opts *UploadOptions) (*Metadata, error) {
	if opts == nil {
		opts = DefaultUploadOptions(fileHeader.Filename)
	}

	// Auto-detect content type if not provided
	if opts.ContentType == "" {
		opts.ContentType = fileHeader.Header.Get("Content-Type")
	}

	// Use filename as key if not provided
	if opts.Key == "" {
		opts.Key = fileHeader.Filename
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open multipart file: %w", err)
	}
	defer src.Close()

	return s.Upload(ctx, src, opts)
}

// Download retrieves a file
func (s *MinioStorage) Download(ctx context.Context, key string, opts *DownloadOptions) (io.ReadCloser, error) {
	if opts == nil {
		opts = DefaultDownloadOptions()
	}

	getOpts := minio.GetObjectOptions{}

	if opts.Range != "" {
		if err := getOpts.SetRange(0, 0); err != nil {
			// Parse range header manually if needed
			s.log.Warn("Failed to set range", zap.Error(err))
		}
	}

	obj, err := s.client.GetObject(ctx, s.config.BucketName, key, getOpts)
	if err != nil {
		s.log.Error("Failed to download", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("failed to download %s: %w", key, err)
	}

	return obj, nil
}

// Delete removes a file
func (s *MinioStorage) Delete(ctx context.Context, key string, opts *DeleteOptions) error {
	if opts == nil {
		opts = DefaultDeleteOptions()
	}

	if opts.Recursive {
		// Delete all objects with the key as prefix
		objectsCh := s.client.ListObjects(ctx, s.config.BucketName, minio.ListObjectsOptions{
			Prefix:    key,
			Recursive: true,
		})

		errorCh := s.client.RemoveObjects(ctx, s.config.BucketName, objectsCh, minio.RemoveObjectsOptions{})

		for err := range errorCh {
			if err.Err != nil {
				s.log.Error("Failed to delete object", zap.String("key", err.ObjectName), zap.Error(err.Err))
				return fmt.Errorf("failed to delete %s: %w", err.ObjectName, err.Err)
			}
		}

		s.log.Info("Deleted recursively", zap.String("prefix", key))
		return nil
	}

	err := s.client.RemoveObject(ctx, s.config.BucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		s.log.Error("Failed to delete", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("failed to delete %s: %w", key, err)
	}

	s.log.Info("Deleted", zap.String("key", key))
	return nil
}

// Exists checks if a file exists
func (s *MinioStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.config.BucketName, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence of %s: %w", key, err)
	}
	return true, nil
}

// GetMetadata retrieves file metadata without downloading
func (s *MinioStorage) GetMetadata(ctx context.Context, key string) (*Metadata, error) {
	stat, err := s.client.StatObject(ctx, s.config.BucketName, key, minio.StatObjectOptions{})
	if err != nil {
		s.log.Error("Failed to get metadata", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("failed to get metadata for %s: %w", key, err)
	}

	fileURL, err := s.buildFileURL(key)
	if err != nil {
		return nil, err
	}

	return &Metadata{
		Key:         key,
		FileName:    path.Base(key),
		ContentType: stat.ContentType,
		URL:         fileURL,
		Size:        stat.Size,
		ETag:        stat.ETag,
		ModTime:     stat.LastModified,
		Custom:      stat.UserMetadata,
	}, nil
}

// List lists files matching criteria
func (s *MinioStorage) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	if opts == nil {
		opts = DefaultListOptions()
	}

	listOpts := minio.ListObjectsOptions{
		Prefix:    opts.Prefix,
		Recursive: true,
		MaxKeys:   opts.MaxResults,
	}

	var items []*Metadata
	objectCh := s.client.ListObjects(ctx, s.config.BucketName, listOpts)

	count := 0
	for object := range objectCh {
		if object.Err != nil {
			s.log.Error("List error", zap.Error(object.Err))
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}

		fileURL, _ := s.buildFileURL(object.Key)

		items = append(items, &Metadata{
			Key:         object.Key,
			FileName:    path.Base(object.Key),
			ContentType: object.ContentType,
			URL:         fileURL,
			Size:        object.Size,
			ETag:        object.ETag,
			ModTime:     object.LastModified,
		})

		count++
		if opts.MaxResults > 0 && count >= opts.MaxResults {
			break
		}
	}

	return &ListResult{
		Items:   items,
		HasMore: false, // MinIO doesn't provide continuation token easily
	}, nil
}

// GetURL returns a public URL for accessing the file
func (s *MinioStorage) GetURL(ctx context.Context, key string) (string, error) {
	return s.buildFileURL(key)
}

// GetSignedURL returns a temporary signed URL
func (s *MinioStorage) GetSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignedURL, err := s.client.PresignedGetObject(ctx, s.config.BucketName, key, expiry, nil)
	if err != nil {
		s.log.Error("Failed to generate signed URL", zap.String("key", key), zap.Error(err))
		return "", fmt.Errorf("failed to generate signed URL for %s: %w", key, err)
	}

	return presignedURL.String(), nil
}

// getImageDimensions extracts width and height of an image
func getImageDimensions(reader io.Reader) (int, int, error) {
	imgConfig, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, err
	}
	return imgConfig.Width, imgConfig.Height, nil
}

// buildFileURL constructs a file URL from the storage key
func (s *MinioStorage) buildFileURL(key string) (string, error) {
	scheme := "http"
	if s.config.UseSSL {
		scheme = "https"
	}

	u, err := url.Parse(fmt.Sprintf("%s://%s", scheme, s.config.Endpoint))
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %w", err)
	}

	u.Path = path.Join(u.Path, s.config.BucketName, key)

	fileURL := u.String()

	// Safety net: fix accidental double protocol
	fileURL = strings.Replace(fileURL, "http://https:/", "http://", 1)
	fileURL = strings.Replace(fileURL, "https://http:/", "https://", 1)

	return fileURL, nil
}
