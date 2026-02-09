package imgutil_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http/httptest"

	"github.com/bronystylecrazy/ultrastructure/imgutil"
)

func ExampleEncodeThumbHashBase64() {
	ctx := context.Background()

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{G: 255, A: 255})
	img.Set(0, 1, color.RGBA{B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	hashB64 := imgutil.EncodeThumbHashBase64(ctx, img)
	fmt.Println(len(hashB64) > 0)
	// Output: true
}

func ExampleEncodeThumbHashBase64FromReader() {
	ctx := context.Background()

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{G: 255, A: 255})
	img.Set(0, 1, color.RGBA{B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)

	hashB64, err := imgutil.EncodeThumbHashBase64FromReader(ctx, bytes.NewReader(buf.Bytes()))
	fmt.Println(err == nil, len(hashB64) > 0)
	// Output: true true
}

func ExampleGenerateThumbHash() {
	ctx := context.Background()

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{G: 255, A: 255})
	img.Set(0, 1, color.RGBA{B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	var imgBuf bytes.Buffer
	_ = png.Encode(&imgBuf, img)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "sample.png")
	_, _ = part.Write(imgBuf.Bytes())
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_ = req.ParseMultipartForm(1 << 20)
	fileHeader := req.MultipartForm.File["file"][0]

	result, err := imgutil.GenerateThumbHash(ctx, fileHeader)
	fmt.Println(err == nil, len(result.HashBase64) > 0, result.Width > 0, result.Height > 0)
	// Output: true true true true
}
