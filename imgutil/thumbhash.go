package imgutil

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"sync"
	"unsafe"

	"github.com/bronystylecrazy/ultrastructure/imgutil/internal/thumbhash"
)

var decodeReaderPool = sync.Pool{
	New: func() any {
		return bufio.NewReaderSize(bytes.NewReader(nil), 4096)
	},
}

type DecodingCfg = thumbhash.DecodingCfg
type Hash = thumbhash.Hash
type ThumbHashResult struct {
	HashBase64 string
	Width      int
	Height     int
	AvgR       float64
	AvgG       float64
	AvgB       float64
	AvgA       float64
}

type DecodedThumbHashResult struct {
	Image         image.Image
	DecodedWidth  int
	DecodedHeight int
	ApproxAvgR    float64
	ApproxAvgG    float64
	ApproxAvgB    float64
	ApproxAvgA    float64
}

func EncodeThumbHash(ctx context.Context, img image.Image) []byte {
	if err := checkContext(ctx); err != nil {
		return nil
	}
	if img == nil {
		return nil
	}
	return thumbhash.EncodeImage(img)
}

func EncodeThumbHashBase64(ctx context.Context, img image.Image) string {
	if err := checkContext(ctx); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(EncodeThumbHash(ctx, img))
}

func EncodeThumbHashResult(ctx context.Context, img image.Image) (ThumbHashResult, error) {
	return BuildThumbHashResult(ctx, img)
}

func EncodeThumbHashFromReader(ctx context.Context, r io.Reader) ([]byte, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("%w: reader is nil", ErrInvalidInput)
	}

	if data, ok := bytesFromReader(r); ok {
		return EncodeThumbHashFromBytes(ctx, data)
	}

	img, err := decodeImage(ctx, r)
	if err != nil {
		return nil, err
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return EncodeThumbHash(ctx, img), nil
}

func EncodeThumbHashBase64FromReader(ctx context.Context, r io.Reader) (string, error) {
	if err := checkContext(ctx); err != nil {
		return "", err
	}
	if r == nil {
		return "", fmt.Errorf("%w: reader is nil", ErrInvalidInput)
	}

	if data, ok := bytesFromReader(r); ok {
		return EncodeThumbHashBase64FromBytes(ctx, data)
	}

	hash, err := EncodeThumbHashFromReader(ctx, r)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

func EncodeThumbHashFromBytes(ctx context.Context, data []byte) ([]byte, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: input bytes are empty", ErrInvalidInput)
	}

	img, err := decodeImage(ctx, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return EncodeThumbHash(ctx, img), nil
}

func EncodeThumbHashResultFromBytes(ctx context.Context, data []byte) (ThumbHashResult, error) {
	if err := checkContext(ctx); err != nil {
		return ThumbHashResult{}, err
	}
	if len(data) == 0 {
		return ThumbHashResult{}, fmt.Errorf("%w: input bytes are empty", ErrInvalidInput)
	}

	img, err := decodeImage(ctx, bytes.NewReader(data))
	if err != nil {
		return ThumbHashResult{}, err
	}
	return BuildThumbHashResult(ctx, img)
}

func EncodeThumbHashBase64FromBytes(ctx context.Context, data []byte) (string, error) {
	hash, err := EncodeThumbHashFromBytes(ctx, data)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

func EncodeThumbHashResultFromReader(ctx context.Context, r io.Reader) (ThumbHashResult, error) {
	if err := checkContext(ctx); err != nil {
		return ThumbHashResult{}, err
	}
	if r == nil {
		return ThumbHashResult{}, fmt.Errorf("%w: reader is nil", ErrInvalidInput)
	}

	if data, ok := bytesFromReader(r); ok {
		return EncodeThumbHashResultFromBytes(ctx, data)
	}

	img, err := decodeImage(ctx, r)
	if err != nil {
		return ThumbHashResult{}, err
	}
	return BuildThumbHashResult(ctx, img)
}

func DecodeThumbHash(ctx context.Context, hashData []byte) (image.Image, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if len(hashData) == 0 {
		return nil, fmt.Errorf("%w: hash is empty", ErrInvalidInput)
	}

	img, err := thumbhash.DecodeImage(hashData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeHash, err)
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return img, nil
}

func DecodeThumbHashFromBytes(ctx context.Context, data []byte) (image.Image, error) {
	return DecodeThumbHash(ctx, data)
}

func DecodeThumbHashFromBytesWithCfg(ctx context.Context, data []byte, cfg DecodingCfg) (image.Image, error) {
	return DecodeThumbHashWithCfg(ctx, data, cfg)
}

func DecodeThumbHashFromReader(ctx context.Context, r io.Reader) (image.Image, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("%w: reader is nil", ErrInvalidInput)
	}
	if data, ok := bytesFromReader(r); ok {
		return DecodeThumbHashFromBytes(ctx, data)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: read hash bytes: %w", ErrInvalidInput, err)
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return DecodeThumbHashFromBytes(ctx, data)
}

func DecodeThumbHashFromReaderWithCfg(ctx context.Context, r io.Reader, cfg DecodingCfg) (image.Image, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("%w: reader is nil", ErrInvalidInput)
	}
	if data, ok := bytesFromReader(r); ok {
		return DecodeThumbHashFromBytesWithCfg(ctx, data, cfg)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: read hash bytes: %w", ErrInvalidInput, err)
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return DecodeThumbHashFromBytesWithCfg(ctx, data, cfg)
}

func DecodeThumbHashWithCfg(ctx context.Context, hashData []byte, cfg DecodingCfg) (image.Image, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if len(hashData) == 0 {
		return nil, fmt.Errorf("%w: hash is empty", ErrInvalidInput)
	}

	img, err := thumbhash.DecodeImageWithCfg(hashData, cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeHash, err)
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return img, nil
}

func DecodeThumbHashBase64(ctx context.Context, hashBase64 string) (image.Image, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if hashBase64 == "" {
		return nil, fmt.Errorf("%w: base64 hash is empty", ErrInvalidInput)
	}

	hashData, err := base64.StdEncoding.DecodeString(hashBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: decode base64 hash: %w", ErrInvalidInput, err)
	}
	return DecodeThumbHash(ctx, hashData)
}

func DecodeThumbHashBase64WithCfg(ctx context.Context, hashBase64 string, cfg DecodingCfg) (image.Image, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if hashBase64 == "" {
		return nil, fmt.Errorf("%w: base64 hash is empty", ErrInvalidInput)
	}

	hashData, err := base64.StdEncoding.DecodeString(hashBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: decode base64 hash: %w", ErrInvalidInput, err)
	}
	return DecodeThumbHashWithCfg(ctx, hashData, cfg)
}

func DecodeThumbHashResult(ctx context.Context, hashData []byte) (DecodedThumbHashResult, error) {
	img, err := DecodeThumbHash(ctx, hashData)
	if err != nil {
		return DecodedThumbHashResult{}, err
	}
	return BuildDecodedThumbHashResult(ctx, img)
}

func DecodeThumbHashResultWithCfg(ctx context.Context, hashData []byte, cfg DecodingCfg) (DecodedThumbHashResult, error) {
	img, err := DecodeThumbHashWithCfg(ctx, hashData, cfg)
	if err != nil {
		return DecodedThumbHashResult{}, err
	}
	return BuildDecodedThumbHashResult(ctx, img)
}

func DecodeHash(ctx context.Context, hashData []byte, cfg *DecodingCfg) (*Hash, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if len(hashData) == 0 {
		return nil, fmt.Errorf("%w: hash is empty", ErrInvalidInput)
	}
	if cfg == nil {
		cfg = &DecodingCfg{}
	}

	var hash Hash
	if err := hash.Decode(hashData, cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeHash, err)
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return &hash, nil
}

func DecodeHashBase64(ctx context.Context, hashBase64 string, cfg *DecodingCfg) (*Hash, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if hashBase64 == "" {
		return nil, fmt.Errorf("%w: base64 hash is empty", ErrInvalidInput)
	}

	hashData, err := base64.StdEncoding.DecodeString(hashBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: decode base64 hash: %w", ErrInvalidInput, err)
	}
	return DecodeHash(ctx, hashData, cfg)
}

func EncodeHash(ctx context.Context, hash *Hash) ([]byte, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if hash == nil {
		return nil, fmt.Errorf("%w: hash is nil", ErrInvalidInput)
	}
	return hash.Encode(), nil
}

func EncodeHashBase64(ctx context.Context, hash *Hash) (string, error) {
	encoded, err := EncodeHash(ctx, hash)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encoded), nil
}

func HashSize(ctx context.Context, hash *Hash, baseSize int) (int, int, error) {
	if err := checkContext(ctx); err != nil {
		return 0, 0, err
	}
	if hash == nil {
		return 0, 0, fmt.Errorf("%w: hash is nil", ErrInvalidInput)
	}
	w, h := hash.Size(baseSize)
	return w, h, nil
}

// GenerateThumbHash is a multipart adapter that returns ThumbHash plus image metadata.
func GenerateThumbHash(ctx context.Context, fileHeader *multipart.FileHeader) (ThumbHashResult, error) {
	if err := checkContext(ctx); err != nil {
		return ThumbHashResult{}, err
	}
	if fileHeader == nil {
		return ThumbHashResult{}, fmt.Errorf("%w: file header is nil", ErrInvalidInput)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return ThumbHashResult{}, fmt.Errorf("%w: %w", ErrOpenFile, err)
	}
	defer file.Close()

	img, err := decodeImage(ctx, file)
	if err != nil {
		return ThumbHashResult{}, err
	}

	return BuildThumbHashResult(ctx, img)
}

func decodeImage(ctx context.Context, r io.Reader) (image.Image, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	br := decodeReaderPool.Get().(*bufio.Reader)
	br.Reset(r)
	defer func() {
		br.Reset(bytes.NewReader(nil))
		decodeReaderPool.Put(br)
	}()

	header, err := br.Peek(8)
	if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
		return nil, fmt.Errorf("%w: %w", ErrDecodeImage, err)
	}

	var img image.Image
	switch {
	case isPNG(header):
		img, err = png.Decode(br)
	case isJPEG(header):
		img, err = jpeg.Decode(br)
	default:
		img, _, err = image.Decode(br)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeImage, err)
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return img, nil
}

func isPNG(header []byte) bool {
	return len(header) >= 8 &&
		header[0] == 0x89 &&
		header[1] == 0x50 &&
		header[2] == 0x4e &&
		header[3] == 0x47 &&
		header[4] == 0x0d &&
		header[5] == 0x0a &&
		header[6] == 0x1a &&
		header[7] == 0x0a
}

func isJPEG(header []byte) bool {
	return len(header) >= 2 && header[0] == 0xff && header[1] == 0xd8
}

func bytesFromReader(r io.Reader) ([]byte, bool) {
	switch v := r.(type) {
	case *bytes.Reader:
		return bytesReaderRemaining(v)
	case *bytes.Buffer:
		if v.Len() == 0 {
			return nil, false
		}
		return v.Bytes(), true
	default:
		return nil, false
	}
}

// bytesReaderShadow mirrors bytes.Reader memory layout for a zero-copy view.
// This avoids an extra allocation/copy when callers pass *bytes.Reader.
type bytesReaderShadow struct {
	s        []byte
	i        int64
	prevRune int
}

func bytesReaderRemaining(r *bytes.Reader) ([]byte, bool) {
	if r == nil {
		return nil, false
	}
	shadow := (*bytesReaderShadow)(unsafe.Pointer(r))
	if shadow.i < 0 || shadow.i > int64(len(shadow.s)) {
		return nil, false
	}
	remaining := shadow.s[shadow.i:]
	if len(remaining) == 0 {
		return nil, false
	}
	return remaining, true
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}
	return nil
}

func BuildThumbHashResult(ctx context.Context, img image.Image) (ThumbHashResult, error) {
	if err := checkContext(ctx); err != nil {
		return ThumbHashResult{}, err
	}
	if img == nil {
		return ThumbHashResult{}, fmt.Errorf("%w: image is nil", ErrInvalidInput)
	}

	hashBytes, meta := thumbhash.EncodeImageWithMeta(img)
	hashBase64 := base64.StdEncoding.EncodeToString(hashBytes)
	if hashBase64 == "" {
		return ThumbHashResult{}, fmt.Errorf("%w: failed to encode thumbhash", ErrInvalidInput)
	}

	return ThumbHashResult{
		HashBase64: hashBase64,
		Width:      meta.Width,
		Height:     meta.Height,
		AvgR:       meta.AvgR,
		AvgG:       meta.AvgG,
		AvgB:       meta.AvgB,
		AvgA:       meta.AvgA,
	}, nil
}

func BuildDecodedThumbHashResult(ctx context.Context, img image.Image) (DecodedThumbHashResult, error) {
	if err := checkContext(ctx); err != nil {
		return DecodedThumbHashResult{}, err
	}
	if img == nil {
		return DecodedThumbHashResult{}, fmt.Errorf("%w: image is nil", ErrInvalidInput)
	}

	bounds := img.Bounds()
	avgR, avgG, avgB, avgA := averageRGBA(img)

	return DecodedThumbHashResult{
		Image:         img,
		DecodedWidth:  bounds.Dx(),
		DecodedHeight: bounds.Dy(),
		ApproxAvgR:    avgR,
		ApproxAvgG:    avgG,
		ApproxAvgB:    avgB,
		ApproxAvgA:    avgA,
	}, nil
}

func averageRGBA(img image.Image) (float64, float64, float64, float64) {
	b := img.Bounds()
	pixels := b.Dx() * b.Dy()
	if pixels <= 0 {
		return 0, 0, 0, 0
	}

	var sumR, sumG, sumB, sumA float64
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			sumR += float64(r) / 257.0
			sumG += float64(g) / 257.0
			sumB += float64(bl) / 257.0
			sumA += float64(a) / 257.0
		}
	}

	div := float64(pixels)
	return sumR / div, sumG / div, sumB / div, sumA / div
}
