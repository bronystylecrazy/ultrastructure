package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/lifecycle"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/storage/s3"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(di.App(
		di.Diagnostics(),
		lifecycle.Module(),
		otel.Module(),
		s3.Module(
			s3.UseOtel(),
			s3.UseInterfaces(),
		),
		web.Module(
			web.UseOtel(),
			di.Provide(NewUploadHandler),
			di.Invoke(RegisterRoutes),
		),
	).Build())

	app.Run()
}

func NewFiberApp() *fiber.App {
	return fiber.New()
}

func RegisterRoutes(app *fiber.App, cfg s3.Config, uploader s3.Uploader, downloader s3.Downloader, presigner s3.Presigner) error {
	if cfg.Bucket == "" {
		return fmt.Errorf("storage.s3.bucket is required")
	}

	// app.Get("/download", func(c fiber.Ctx) error {
	// 	key := c.Query("key")
	// 	if key == "" {
	// 		return fiber.NewError(fiber.StatusBadRequest, "missing key")
	// 	}

	// 	out, err := downloader.GetObject(c.Context(), &s3sdk.GetObjectInput{
	// 		Bucket: aws.String(cfg.Bucket),
	// 		Key:    aws.String(key),
	// 	})
	// 	if err != nil {
	// 		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("download failed: %v", err))
	// 	}
	// 	defer out.Body.Close()

	// 	if out.ContentType != nil {
	// 		c.Set("Content-Type", *out.ContentType)
	// 	}
	// 	if out.ContentLength != nil {
	// 		c.Set("Content-Length", fmt.Sprintf("%d", out.ContentLength))
	// 	}
	// 	c.Status(fiber.StatusOK)
	// 	return c.SendStream(out.Body)
	// })

	// app.Get("/presign", func(c fiber.Ctx) error {
	// 	key := c.Query("key")
	// 	if key == "" {
	// 		return fiber.NewError(fiber.StatusBadRequest, "missing key")
	// 	}

	// 	req, err := presigner.PresignGetObject(c.Context(), &s3sdk.GetObjectInput{
	// 		Bucket: aws.String(cfg.Bucket),
	// 		Key:    aws.String(key),
	// 	}, func(opts *s3sdk.PresignOptions) {
	// 		opts.Expires = 15 * time.Minute
	// 	})
	// 	if err != nil {
	// 		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("presign failed: %v", err))
	// 	}

	// 	return c.Type("text").SendString(req.URL)
	// })

	return nil
}

func RunServer(lc fx.Lifecycle, app *fiber.App) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Println("listening on :8080")
				if err := app.Listen(":8080"); err != nil {
					log.Println("server error:", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return app.ShutdownWithTimeout(5 * time.Second)
		},
	})
}
