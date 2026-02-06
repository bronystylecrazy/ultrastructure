package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/lifecycle"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/storage/s3"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(di.App(
		otel.Module(),
		lifecycle.Module(),
		s3.Module(
			s3.UseInterfaces(),
		),
		di.Provide(NewServer),
		di.Invoke(RunServer),
	).Build())

	app.Run()
}

func NewServer(cfg s3.Config, uploader s3.Uploader, downloader s3.Downloader, presigner s3.Presigner) (*http.Server, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("storage.s3.bucket is required")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "missing key", http.StatusBadRequest)
			return
		}

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "invalid multipart", http.StatusBadRequest)
			return
		}

		file, fh, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		file.Close()

		out, err := s3.UploadFileHeader(r.Context(), uploader, cfg.Bucket, key, fh)
		if err != nil {
			http.Error(w, fmt.Sprintf("upload failed: %v", err), http.StatusInternalServerError)
			return
		}

		etag := ""
		if out != nil && out.ETag != nil {
			etag = *out.ETag
		}

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "uploaded key=%s etag=%s\n", key, etag)
	})

	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "missing key", http.StatusBadRequest)
			return
		}

		out, err := downloader.GetObject(r.Context(), &s3sdk.GetObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("download failed: %v", err), http.StatusInternalServerError)
			return
		}
		defer out.Body.Close()

		if out.ContentType != nil {
			w.Header().Set("Content-Type", *out.ContentType)
		}
		if out.ContentLength != nil {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", out.ContentLength))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, out.Body)
	})

	mux.HandleFunc("/presign", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "missing key", http.StatusBadRequest)
			return
		}

		req, err := presigner.PresignGetObject(r.Context(), &s3sdk.GetObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    aws.String(key),
		}, func(opts *s3sdk.PresignOptions) {
			opts.Expires = 15 * time.Minute
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("presign failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, req.URL)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return server, nil
}

func RunServer(lc fx.Lifecycle, server *http.Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Println("listening on", server.Addr)
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Println("server error:", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return server.Shutdown(shutdownCtx)
		},
	})
}
