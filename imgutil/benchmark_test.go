package imgutil

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func BenchmarkEncodeThumbHash(b *testing.B) {
	img := benchmarkFixtureImage()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeThumbHash(ctx, img)
	}
}

func BenchmarkEncodeThumbHashBase64(b *testing.B) {
	img := benchmarkFixtureImage()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeThumbHashBase64(ctx, img)
	}
}

func BenchmarkEncodeThumbHashFromReader(b *testing.B) {
	data := benchmarkPNGData(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeThumbHashFromReader(ctx, bytes.NewReader(data))
	}
}

func BenchmarkEncodeThumbHashFromBytes(b *testing.B) {
	data := benchmarkPNGData(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeThumbHashFromBytes(ctx, data)
	}
}

func BenchmarkEncodeThumbHashBase64FromReader(b *testing.B) {
	data := benchmarkPNGData(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeThumbHashBase64FromReader(ctx, bytes.NewReader(data))
	}
}

func BenchmarkEncodeThumbHashBase64FromBytes(b *testing.B) {
	data := benchmarkPNGData(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeThumbHashBase64FromBytes(ctx, data)
	}
}

func BenchmarkEncodeThumbHashResult(b *testing.B) {
	img := benchmarkFixtureImage()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeThumbHashResult(ctx, img)
	}
}

func BenchmarkEncodeThumbHashResultFromBytes(b *testing.B) {
	data := benchmarkPNGData(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeThumbHashResultFromBytes(ctx, data)
	}
}

func BenchmarkEncodeThumbHashResultFromReader(b *testing.B) {
	data := benchmarkPNGData(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeThumbHashResultFromReader(ctx, bytes.NewReader(data))
	}
}

func BenchmarkGenerateThumbHash(b *testing.B) {
	data := benchmarkPNGData(b)
	header := benchmarkMultipartFileHeader(b, "file", "bench.png", data)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateThumbHash(ctx, header)
	}
}

func BenchmarkDecodeThumbHash(b *testing.B) {
	ctx := context.Background()
	hash := EncodeThumbHash(ctx, benchmarkFixtureImage())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeThumbHash(ctx, hash)
	}
}

func BenchmarkDecodeThumbHashFromBytes(b *testing.B) {
	ctx := context.Background()
	hash := EncodeThumbHash(ctx, benchmarkFixtureImage())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeThumbHashFromBytes(ctx, hash)
	}
}

func BenchmarkDecodeThumbHashFromReader(b *testing.B) {
	ctx := context.Background()
	hash := EncodeThumbHash(ctx, benchmarkFixtureImage())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeThumbHashFromReader(ctx, bytes.NewReader(hash))
	}
}

func BenchmarkDecodeThumbHashBase64(b *testing.B) {
	ctx := context.Background()
	hashB64 := EncodeThumbHashBase64(ctx, benchmarkFixtureImage())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeThumbHashBase64(ctx, hashB64)
	}
}

func BenchmarkDecodeHash(b *testing.B) {
	ctx := context.Background()
	hash := EncodeThumbHash(ctx, benchmarkFixtureImage())
	cfg := &DecodingCfg{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeHash(ctx, hash, cfg)
	}
}

func BenchmarkEncodeHash(b *testing.B) {
	ctx := context.Background()
	hashBytes := EncodeThumbHash(ctx, benchmarkFixtureImage())
	h, _ := DecodeHash(ctx, hashBytes, &DecodingCfg{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeHash(ctx, h)
	}
}

func benchmarkFixtureImage() image.Image {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 3) % 256),
				G: uint8((y * 5) % 256),
				B: uint8(((x + y) * 7) % 256),
				A: 255,
			})
		}
	}
	return img
}

func benchmarkPNGData(b *testing.B) []byte {
	b.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, benchmarkFixtureImage()); err != nil {
		b.Fatalf("encode benchmark png: %v", err)
	}
	return buf.Bytes()
}

func benchmarkMultipartFileHeader(b *testing.B, fieldName string, fileName string, data []byte) *multipart.FileHeader {
	b.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		b.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		b.Fatalf("write form data: %v", err)
	}
	if err := writer.Close(); err != nil {
		b.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(int64(len(data)) + 1024); err != nil {
		b.Fatalf("parse multipart form: %v", err)
	}

	fileHeaders := req.MultipartForm.File[fieldName]
	if len(fileHeaders) != 1 {
		b.Fatalf("unexpected file headers count: got %d want %d", len(fileHeaders), 1)
	}
	return fileHeaders[0]
}
