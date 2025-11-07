package imageutil

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"

	"go.n16f.net/thumbhash"
)

// GenerateThumbHash takes a multipart.FileHeader, decodes the image, generates a ThumbHash, and returns its base64 string representation.
func GenerateThumbHash(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	imgBytes := bytes.NewBuffer(nil)
	if _, err := imgBytes.ReadFrom(file); err != nil {
		return "", fmt.Errorf("failed to read file into buffer: %w", err)
	}

	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(imgBytes.Bytes()))
	if err != nil {
		// Try decoding as JPEG if initial decode fails (common for some uploads)
		img, err = jpeg.Decode(bytes.NewReader(imgBytes.Bytes()))
		if err != nil {
			// Try decoding as PNG if JPEG also fails
			img, err = png.Decode(bytes.NewReader(imgBytes.Bytes()))
			if err != nil {
				return "", fmt.Errorf("failed to decode image: %w", err)
			}
		}
	}

	// Generate ThumbHash
	hash := thumbhash.EncodeImage(img)
	thumbHashBase64 := base64.StdEncoding.EncodeToString(hash)

	return thumbHashBase64, nil
}
