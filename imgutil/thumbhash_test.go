package imgutil

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEncodeThumbHashFromReaderValidFormatsDeterministic(t *testing.T) {
	img := testFixtureImage()

	formats := []struct {
		name string
		data []byte
	}{
		{name: "png", data: encodePNG(t, img)},
		{name: "jpeg", data: encodeJPEG(t, img)},
	}

	for _, tc := range formats {
		t.Run(tc.name, func(t *testing.T) {
			hash1, err := EncodeThumbHashFromReader(context.Background(), bytes.NewReader(tc.data))
			if err != nil {
				t.Fatalf("encode thumbhash from reader: %v", err)
			}
			hash2, err := EncodeThumbHashFromReader(context.Background(), bytes.NewReader(tc.data))
			if err != nil {
				t.Fatalf("encode thumbhash from reader again: %v", err)
			}
			if len(hash1) == 0 {
				t.Fatal("thumbhash is empty")
			}
			if !bytes.Equal(hash1, hash2) {
				t.Fatal("thumbhash is not deterministic for the same input")
			}

			b641, err := EncodeThumbHashBase64FromReader(context.Background(), bytes.NewReader(tc.data))
			if err != nil {
				t.Fatalf("encode base64 thumbhash from reader: %v", err)
			}
			b642, err := EncodeThumbHashBase64FromReader(context.Background(), bytes.NewReader(tc.data))
			if err != nil {
				t.Fatalf("encode base64 thumbhash from reader again: %v", err)
			}
			if b641 == "" {
				t.Fatal("base64 thumbhash is empty")
			}
			if b641 != b642 {
				t.Fatal("base64 thumbhash is not deterministic for the same input")
			}
		})
	}
}

func TestEncodeThumbHashFromReaderInvalidData(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "garbage", data: []byte("not-an-image")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EncodeThumbHashFromReader(context.Background(), bytes.NewReader(tc.data))
			if err == nil {
				t.Fatal("expected decode error")
			}
			if !errors.Is(err, ErrDecodeImage) {
				t.Fatalf("expected ErrDecodeImage, got: %v", err)
			}
		})
	}
}

func TestEncodeThumbHashImageDeterministic(t *testing.T) {
	img := testFixtureImage()
	hash1 := EncodeThumbHash(context.Background(), img)
	hash2 := EncodeThumbHash(context.Background(), img)
	if len(hash1) == 0 {
		t.Fatal("thumbhash is empty")
	}
	if !bytes.Equal(hash1, hash2) {
		t.Fatal("thumbhash is not deterministic")
	}

	b641 := EncodeThumbHashBase64(context.Background(), img)
	b642 := EncodeThumbHashBase64(context.Background(), img)
	if b641 == "" {
		t.Fatal("base64 thumbhash is empty")
	}
	if b641 != b642 {
		t.Fatal("base64 thumbhash is not deterministic")
	}
}

func TestDecodeThumbHashRoundTrip(t *testing.T) {
	img := testFixtureImage()
	hash := EncodeThumbHash(context.Background(), img)

	decoded, err := DecodeThumbHash(context.Background(), hash)
	if err != nil {
		t.Fatalf("decode thumbhash: %v", err)
	}
	if decoded == nil {
		t.Fatal("decoded image is nil")
	}
	if decoded.Bounds().Dx() <= 0 || decoded.Bounds().Dy() <= 0 {
		t.Fatalf("decoded image has invalid size: %v", decoded.Bounds())
	}
}

func TestDecodeThumbHashWithCfgRoundTrip(t *testing.T) {
	img := testFixtureImage()
	hash := EncodeThumbHash(context.Background(), img)

	cfg := DecodingCfg{
		BaseSize:        24,
		SaturationBoost: 1.5,
	}
	decoded, err := DecodeThumbHashWithCfg(context.Background(), hash, cfg)
	if err != nil {
		t.Fatalf("decode thumbhash with cfg: %v", err)
	}
	if decoded == nil {
		t.Fatal("decoded image is nil")
	}
	if decoded.Bounds().Dx() <= 0 || decoded.Bounds().Dy() <= 0 {
		t.Fatalf("decoded image has invalid size: %v", decoded.Bounds())
	}
}

func TestDecodeThumbHashErrors(t *testing.T) {
	_, err := DecodeThumbHash(context.Background(), nil)
	if err == nil {
		t.Fatal("expected invalid input error")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}

	_, err = DecodeThumbHash(context.Background(), []byte("not-a-thumbhash"))
	if err == nil {
		t.Fatal("expected decode hash error")
	}
	if !errors.Is(err, ErrDecodeHash) {
		t.Fatalf("expected ErrDecodeHash, got: %v", err)
	}
	if !errors.Is(err, ErrInvalidHash) {
		t.Fatalf("expected ErrInvalidHash, got: %v", err)
	}
}

func TestDecodeThumbHashBase64RoundTrip(t *testing.T) {
	img := testFixtureImage()
	hashB64 := EncodeThumbHashBase64(context.Background(), img)

	decoded, err := DecodeThumbHashBase64(context.Background(), hashB64)
	if err != nil {
		t.Fatalf("decode base64 thumbhash: %v", err)
	}
	if decoded == nil {
		t.Fatal("decoded image is nil")
	}
	if decoded.Bounds().Dx() <= 0 || decoded.Bounds().Dy() <= 0 {
		t.Fatalf("decoded image has invalid size: %v", decoded.Bounds())
	}
}

func TestDecodeThumbHashBase64WithCfgRoundTrip(t *testing.T) {
	img := testFixtureImage()
	hashB64 := EncodeThumbHashBase64(context.Background(), img)

	cfg := DecodingCfg{
		BaseSize:        20,
		SaturationBoost: 1.35,
	}
	decoded, err := DecodeThumbHashBase64WithCfg(context.Background(), hashB64, cfg)
	if err != nil {
		t.Fatalf("decode base64 thumbhash with cfg: %v", err)
	}
	if decoded == nil {
		t.Fatal("decoded image is nil")
	}
	if decoded.Bounds().Dx() <= 0 || decoded.Bounds().Dy() <= 0 {
		t.Fatalf("decoded image has invalid size: %v", decoded.Bounds())
	}
}

func TestDecodeThumbHashBase64Errors(t *testing.T) {
	_, err := DecodeThumbHashBase64(context.Background(), "")
	if err == nil {
		t.Fatal("expected invalid input error for empty base64")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}

	_, err = DecodeThumbHashBase64(context.Background(), "%%%")
	if err == nil {
		t.Fatal("expected invalid input error for malformed base64")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestDecodeThumbHashResultRoundTrip(t *testing.T) {
	img := testFixtureImage()
	hash := EncodeThumbHash(context.Background(), img)

	result, err := DecodeThumbHashResult(context.Background(), hash)
	if err != nil {
		t.Fatalf("decode thumbhash result: %v", err)
	}
	if result.Image == nil {
		t.Fatal("decoded result image is nil")
	}
	if result.DecodedWidth <= 0 || result.DecodedHeight <= 0 {
		t.Fatalf("invalid decoded dimensions: %dx%d", result.DecodedWidth, result.DecodedHeight)
	}
	if result.ApproxAvgA <= 0 {
		t.Fatalf("unexpected decoded avg alpha: %f", result.ApproxAvgA)
	}
}

func TestDecodeThumbHashResultWithCfgRoundTrip(t *testing.T) {
	img := testFixtureImage()
	hash := EncodeThumbHash(context.Background(), img)
	cfg := DecodingCfg{
		BaseSize:        40,
		SaturationBoost: 1.2,
	}

	result, err := DecodeThumbHashResultWithCfg(context.Background(), hash, cfg)
	if err != nil {
		t.Fatalf("decode thumbhash result with cfg: %v", err)
	}
	if result.Image == nil {
		t.Fatal("decoded result image is nil")
	}
	if result.DecodedWidth <= 0 || result.DecodedHeight <= 0 {
		t.Fatalf("invalid decoded dimensions: %dx%d", result.DecodedWidth, result.DecodedHeight)
	}
}

func TestDecodeThumbHashFromBytesAndReaderRoundTrip(t *testing.T) {
	img := testFixtureImage()
	hash := EncodeThumbHash(context.Background(), img)

	fromBytes, err := DecodeThumbHashFromBytes(context.Background(), hash)
	if err != nil {
		t.Fatalf("decode from bytes: %v", err)
	}
	if fromBytes == nil || fromBytes.Bounds().Dx() <= 0 || fromBytes.Bounds().Dy() <= 0 {
		t.Fatalf("invalid decoded image from bytes: %#v", fromBytes)
	}

	fromReader, err := DecodeThumbHashFromReader(context.Background(), bytes.NewReader(hash))
	if err != nil {
		t.Fatalf("decode from reader: %v", err)
	}
	if fromReader == nil || fromReader.Bounds().Dx() <= 0 || fromReader.Bounds().Dy() <= 0 {
		t.Fatalf("invalid decoded image from reader: %#v", fromReader)
	}
}

func TestDecodeThumbHashFromReaderErrors(t *testing.T) {
	_, err := DecodeThumbHashFromReader(context.Background(), nil)
	if err == nil {
		t.Fatal("expected invalid input error for nil reader")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestEncodeThumbHashFromReaderContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	img := testFixtureImage()
	data := encodePNG(t, img)
	_, err := EncodeThumbHashFromReader(ctx, bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected canceled context error")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestDecodeThumbHashContextDeadlineExceeded(t *testing.T) {
	img := testFixtureImage()
	hash := EncodeThumbHash(context.Background(), img)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	_, err := DecodeThumbHash(ctx, hash)
	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestGenerateThumbHashContextCanceled(t *testing.T) {
	img := testFixtureImage()
	pngData := encodePNG(t, img)
	header := newMultipartFileHeader(t, "file", "fixture.png", pngData)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := GenerateThumbHash(ctx, header)
	if err == nil {
		t.Fatal("expected canceled context error")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestHashRoundTrip(t *testing.T) {
	img := testFixtureImage()
	hashBytes := EncodeThumbHash(context.Background(), img)

	cfg := &DecodingCfg{
		BaseSize:        24,
		SaturationBoost: 1.4,
	}
	hash, err := DecodeHash(context.Background(), hashBytes, cfg)
	if err != nil {
		t.Fatalf("decode hash: %v", err)
	}
	if hash == nil {
		t.Fatal("decoded hash is nil")
	}

	reEncoded, err := EncodeHash(context.Background(), hash)
	if err != nil {
		t.Fatalf("encode hash: %v", err)
	}
	if len(reEncoded) == 0 {
		t.Fatal("re-encoded hash is empty")
	}

	w, h, err := HashSize(context.Background(), hash, 32)
	if err != nil {
		t.Fatalf("hash size: %v", err)
	}
	if w <= 0 || h <= 0 {
		t.Fatalf("invalid size from hash: %dx%d", w, h)
	}
}

func TestHashBase64RoundTrip(t *testing.T) {
	img := testFixtureImage()
	hashB64 := EncodeThumbHashBase64(context.Background(), img)

	hash, err := DecodeHashBase64(context.Background(), hashB64, nil)
	if err != nil {
		t.Fatalf("decode hash base64: %v", err)
	}
	if hash == nil {
		t.Fatal("decoded hash is nil")
	}

	encodedB64, err := EncodeHashBase64(context.Background(), hash)
	if err != nil {
		t.Fatalf("encode hash base64: %v", err)
	}
	if encodedB64 == "" {
		t.Fatal("encoded hash base64 is empty")
	}
}

func TestHashHelpersErrors(t *testing.T) {
	_, err := DecodeHash(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected invalid input error for empty hash")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}

	_, err = DecodeHash(context.Background(), []byte("not-a-thumbhash"), nil)
	if err == nil {
		t.Fatal("expected decode hash error")
	}
	if !errors.Is(err, ErrDecodeHash) {
		t.Fatalf("expected ErrDecodeHash, got: %v", err)
	}
	if !errors.Is(err, ErrInvalidHash) {
		t.Fatalf("expected ErrInvalidHash, got: %v", err)
	}

	_, err = DecodeHashBase64(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected invalid input error for empty base64 hash")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}

	_, err = EncodeHash(context.Background(), nil)
	if err == nil {
		t.Fatal("expected invalid input error for nil hash")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}

	_, err = EncodeHashBase64(context.Background(), nil)
	if err == nil {
		t.Fatal("expected invalid input error for nil hash")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}

	_, _, err = HashSize(context.Background(), nil, 32)
	if err == nil {
		t.Fatal("expected invalid input error for nil hash")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestGenerateThumbHashAdapterSuccess(t *testing.T) {
	img := testFixtureImage()
	pngData := encodePNG(t, img)

	header := newMultipartFileHeader(t, "file", "fixture.png", pngData)
	got, err := GenerateThumbHash(context.Background(), header)
	if err != nil {
		t.Fatalf("generate thumbhash: %v", err)
	}

	want, err := EncodeThumbHashBase64FromReader(context.Background(), bytes.NewReader(pngData))
	if err != nil {
		t.Fatalf("encode from reader: %v", err)
	}
	if got.HashBase64 != want {
		t.Fatalf("unexpected adapter hash: got %q want %q", got.HashBase64, want)
	}
	if got.Width != 8 || got.Height != 8 {
		t.Fatalf("unexpected dimensions: got %dx%d want %dx%d", got.Width, got.Height, 8, 8)
	}
	if got.AvgA <= 0 {
		t.Fatalf("unexpected avg alpha: %f", got.AvgA)
	}
}

func TestEncodeThumbHashResultFromReader(t *testing.T) {
	img := testFixtureImage()
	pngData := encodePNG(t, img)

	got, err := EncodeThumbHashResultFromReader(context.Background(), bytes.NewReader(pngData))
	if err != nil {
		t.Fatalf("encode thumbhash result from reader: %v", err)
	}
	if got.HashBase64 == "" {
		t.Fatal("empty hash base64")
	}
	if got.Width != 8 || got.Height != 8 {
		t.Fatalf("unexpected dimensions: got %dx%d want %dx%d", got.Width, got.Height, 8, 8)
	}
	if got.AvgA <= 0 {
		t.Fatalf("unexpected avg alpha: %f", got.AvgA)
	}
}

func TestGenerateThumbHashAdapterOpenError(t *testing.T) {
	_, err := GenerateThumbHash(context.Background(), &multipart.FileHeader{})
	if err == nil {
		t.Fatal("expected open error")
	}
	if !errors.Is(err, ErrOpenFile) {
		t.Fatalf("expected ErrOpenFile, got: %v", err)
	}
}

func testFixtureImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 32),
				G: uint8(y * 32),
				B: uint8((x + y) * 16),
				A: 255,
			})
		}
	}
	return img
}

func newMultipartFileHeader(t *testing.T, fieldName string, fileName string, data []byte) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write form file data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(int64(len(data)) + 1024); err != nil {
		t.Fatalf("parse multipart form: %v", err)
	}

	fileHeaders := req.MultipartForm.File[fieldName]
	if len(fileHeaders) != 1 {
		t.Fatalf("unexpected file headers count: got %d want %d", len(fileHeaders), 1)
	}
	return fileHeaders[0]
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png fixture: %v", err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	return buf.Bytes()
}
