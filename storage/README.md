# Storage API Usage Examples

The refactored storage API provides a simple, flexible, and customizable interface for file storage operations.

## Quick Start

### Basic Upload

```go
import (
    "context"
    "bytes"
    "github.com/bronystylecrazy/ultrastructure/src/storage"
)

// Initialize storage
client, err := storage.NewMinioClient(config, logger)
if err != nil {
    log.Fatal(err)
}
store := storage.NewMinioStorage(config, client, logger)

// Simple upload with helper
data := []byte("hello world")
metadata, err := storage.UploadBytes(ctx, store, "path/to/file.txt", data, "text/plain")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Uploaded to: %s\n", metadata.URL)
```

### Upload from Multipart Form

```go
// In your HTTP handler
func uploadHandler(c *fiber.Ctx) error {
    fileHeader, err := c.FormFile("file")
    if err != nil {
        return err
    }

    // Simple upload
    opts := storage.DefaultUploadOptions("uploads/"+fileHeader.Filename)
    metadata, err := store.UploadMultipart(c.Context(), fileHeader, opts)
    if err != nil {
        return err
    }

    return c.JSON(metadata)
}
```

## Advanced Usage

### Upload with Custom Options

```go
// Upload with full control
opts := &storage.UploadOptions{
    Key:                    "images/profile.jpg",
    ContentType:            "image/jpeg",
    ExtractImageDimensions: true,  // Automatically extract width/height
    CacheControl:           "max-age=31536000, public",
    Public:                 true,
    Metadata: map[string]string{
        "user-id":     "12345",
        "upload-date": time.Now().Format(time.RFC3339),
    },
}

file, _ := os.Open("profile.jpg")
defer file.Close()

metadata, err := store.Upload(ctx, file, opts)
```

### Download Operations

```go
// Simple download
reader, err := store.Download(ctx, "path/to/file.txt", nil)
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// Download to bytes (convenience helper)
data, err := storage.DownloadToBytes(ctx, store, "path/to/file.txt")
if err != nil {
    log.Fatal(err)
}

// Download with range
opts := &storage.DownloadOptions{
    Range: "bytes=0-1023", // First 1KB
}
reader, err = store.Download(ctx, "large-file.bin", opts)
```

### File Management

```go
// Check if file exists
exists, err := store.Exists(ctx, "path/to/file.txt")
if exists {
    fmt.Println("File exists!")
}

// Get metadata without downloading
meta, err := store.GetMetadata(ctx, "path/to/file.txt")
fmt.Printf("Size: %d bytes, Modified: %s\n", meta.Size, meta.ModTime)

// Delete single file
err = store.Delete(ctx, "path/to/file.txt", nil)

// Delete recursively (all files with prefix)
err = store.Delete(ctx, "uploads/2024/", &storage.DeleteOptions{
    Recursive: true,
})

// Or use helper
err = storage.DeleteRecursive(ctx, store, "uploads/2024/")
```

### Listing Files

```go
// List all files with prefix
opts := &storage.ListOptions{
    Prefix:     "uploads/images/",
    MaxResults: 100,
}

result, err := store.List(ctx, opts)
if err != nil {
    log.Fatal(err)
}

for _, item := range result.Items {
    fmt.Printf("%s - %d bytes\n", item.Key, item.Size)
}

// Or use helper for simple listing
items, err := storage.ListByPrefix(ctx, store, "uploads/")
```

### URL Generation

```go
// Get public URL
url, err := store.GetURL(ctx, "public/image.jpg")
fmt.Println("Public URL:", url)

// Generate temporary signed URL (expires in 1 hour)
signedURL, err := store.GetSignedURL(ctx, "private/document.pdf", time.Hour)
fmt.Println("Temporary URL:", signedURL)

// Or use helper
url, err = storage.GetTemporaryURL(ctx, store, "private/file.txt", 30*time.Minute)
```

## Helper Functions

The API includes many helper functions for common operations:

```go
// Upload helpers
storage.UploadFile(ctx, store, key, reader, contentType)
storage.UploadBytes(ctx, store, key, data, contentType)
storage.UploadPublic(ctx, store, key, reader, contentType)
storage.UploadWithCache(ctx, store, key, reader, contentType, "max-age=3600")
storage.UploadWithMetadata(ctx, store, key, reader, contentType, metadata)

// Download helpers
storage.DownloadToBytes(ctx, store, key)

// Delete helpers
storage.DeleteSimple(ctx, store, key)
storage.DeleteRecursive(ctx, store, prefix)

// Copy file
storage.CopyFile(ctx, store, srcKey, destKey)

// List helpers
storage.ListAll(ctx, store)
storage.ListByPrefix(ctx, store, prefix)

// Check helpers
storage.FileExists(ctx, store, key)
storage.GetFileInfo(ctx, store, key)

// URL helpers
storage.GetPublicURL(ctx, store, key)
storage.GetTemporaryURL(ctx, store, key, duration)

// Key generation helpers
key := storage.GenerateKey("uploads", "photo.jpg")
// uploads/photo_1704067200.jpg

key = storage.GenerateUniqueKey("images", "avatar.png")
// images/1704067200000000_file.png
```

## Image Upload Example

```go
func uploadImageHandler(c *fiber.Ctx) error {
    fileHeader, err := c.FormFile("image")
    if err != nil {
        return err
    }

    // Generate unique key
    key := storage.GenerateKey("images", fileHeader.Filename)

    // Upload with image dimension extraction
    opts := &storage.UploadOptions{
        Key:                    key,
        ExtractImageDimensions: true, // Extract width/height automatically
        CacheControl:           "max-age=31536000, public",
        Public:                 true,
        Metadata: map[string]string{
            "uploader": c.Locals("user_id").(string),
        },
    }

    metadata, err := store.UploadMultipart(c.Context(), fileHeader, opts)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.JSON(fiber.Map{
        "url":    metadata.URL,
        "width":  metadata.Width,
        "height": metadata.Height,
        "size":   metadata.Size,
        "key":    metadata.Key,
    })
}
```

## Migration from Old API

### Before (Old API)

```go
// Old fragmented interface
metadata, err := uploader.UploadBuffer(ctx, key, reader, size, contentType)
metadata, err := uploader.UploadMultipartFile(ctx, key, fileHeader)
err = deleter.DeleteFile(ctx, key)
reader, err := downloader.DownloadFile(ctx, key)
```

### After (New API)

```go
// New unified interface
opts := storage.DefaultUploadOptions(key)
opts.ContentType = contentType
metadata, err := store.Upload(ctx, reader, opts)

metadata, err := store.UploadMultipart(ctx, fileHeader, opts)
err = store.Delete(ctx, key, nil)
reader, err := store.Download(ctx, key, nil)

// Or use helpers for even simpler code
metadata, err := storage.UploadFile(ctx, store, key, reader, contentType)
err = storage.DeleteSimple(ctx, store, key)
data, err := storage.DownloadToBytes(ctx, store, key)
```

## Streaming Files with Fiber

The storage package includes built-in support for streaming files through Fiber endpoints.

### Simple File Serving

```go
func serveFile(c *fiber.Ctx) error {
    key := c.Params("key")
    return storage.ServeFile(c, store, key)
}

app.Get("/files/:key", serveFile)
```

### Advanced Proxy with Range Requests

```go
func proxyFile(c *fiber.Ctx) error {
    key := c.Params("key")

    // Full control over proxy options
    opts := &storage.ProxyOptions{
        EnableCaching:       true,  // ETag-based caching
        EnableRangeRequests: true,  // Support for range requests (video streaming)
        CacheControl:        "public, max-age=86400",
        Attachment:          false, // Display inline, not download
    }

    return storage.ProxyToFiber(c, store, key, opts)
}

app.Get("/proxy/:key", proxyFile)
```

### Force Download

```go
func downloadFile(c *fiber.Ctx) error {
    key := c.Params("key")
    return storage.DownloadFile(c, store, key)
}

app.Get("/download/:key", downloadFile)
```

### Stream with Custom Headers

```go
func streamVideo(c *fiber.Ctx) error {
    key := c.Params("key")

    // Stream video with range request support for seeking
    return storage.ProxyToFiber(c, store, key, &storage.ProxyOptions{
        EnableRangeRequests: true,
        CacheControl:        "public, max-age=3600",
    })
}

app.Get("/videos/:key", streamVideo)
```

### Direct Stream Access

```go
func customStream(c *fiber.Ctx) error {
    key := c.Params("key")

    // Get reader for custom streaming logic
    reader, err := store.Download(c.Context(), key, nil)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    defer reader.Close()

    // Custom headers
    c.Set("Content-Type", "video/mp4")
    c.Set("Cache-Control", "no-cache")

    return c.SendStream(reader)
}
```

## Key Benefits

1. **Simpler Interface**: Single `Storage` interface instead of multiple fragmented interfaces
2. **Options Pattern**: Flexible configuration without breaking changes
3. **Helper Functions**: Convenience methods for common operations
4. **Better Defaults**: Sensible defaults that work out of the box
5. **More Features**: Built-in support for metadata, caching, signed URLs, listing, streaming, etc.
6. **Easier to Mock**: Single interface to mock for testing
7. **Fiber Integration**: Built-in streaming support with range requests and caching
8. **Better Documentation**: Clear, comprehensive examples

## Type Reference

```go
// Main types
type Storage interface { ... }
type Metadata struct { ... }
type UploadOptions struct { ... }
type DownloadOptions struct { ... }
type DeleteOptions struct { ... }
type ListOptions struct { ... }
type ListResult struct { ... }

// Helper functions
func DefaultUploadOptions(key string) *UploadOptions
func DefaultDownloadOptions() *DownloadOptions
func DefaultDeleteOptions() *DeleteOptions
func DefaultListOptions() *ListOptions
```
