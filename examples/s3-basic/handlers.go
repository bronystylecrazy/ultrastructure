package main

import (
	"fmt"
	"log"

	"github.com/bronystylecrazy/ultrastructure/storage/s3"
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace"
)

type uploadHandler struct {
	uploader s3.Uploader
	config   s3.Config
}

func NewUploadHandler(config s3.Config, uploader s3.Uploader) *uploadHandler {
	return &uploadHandler{uploader: uploader, config: config}
}

func (h *uploadHandler) Handle(r fiber.Router) {
	r.Post("/upload", h.Upload)
}

func (h *uploadHandler) Upload(c fiber.Ctx) error {
	span := trace.SpanFromContext(c.Context())
	log.Println("span valid:", span.SpanContext().IsValid())

	key := c.Query("key")
	if key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing key")
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing file")
	}

	out, err := s3.UploadFileHeader(c.Context(), h.uploader, h.config.Bucket, key, fh)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("upload failed: %v", err))
	}

	etag := ""
	if out != nil && out.ETag != nil {
		etag = *out.ETag
	}

	return c.Type("text").SendString(fmt.Sprintf("uploaded key=%s etag=%s\n", key, etag))
}
