//go:build integration

package testkit

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bronystylecrazy/ultrastructure/caching/rd"
	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/ditest"
	"github.com/bronystylecrazy/ultrastructure/lifecycle"
	"github.com/bronystylecrazy/ultrastructure/otel"
	uss3 "github.com/bronystylecrazy/ultrastructure/storage/s3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type e2eDeps struct {
	db            *gorm.DB
	redisClient   rd.RedisClient
	bucketManager uss3.BucketManager
	uploader      uss3.Uploader
	downloader    uss3.Downloader
	deleter       uss3.Deleter
}

func TestE2EScenarios_PostgresRedisMinIO(t *testing.T) {
	RequireIntegration(t)

	suite := NewSuite(t)
	ctx := suite.Context()

	pg := suite.StartPostgres(PostgresOptions{})
	redisCtr := suite.StartRedis(RedisOptions{})
	minio := suite.StartMinIO(MinIOOptions{})

	bucket := "it-" + uuid.NewString()[:12]
	deps := startE2EApp(t, pg.URL(), redisCtr, minio, bucket)
	mustCreateSchema(t, ctx, deps.db)
	mustCreateBucket(t, ctx, deps.bucketManager, bucket)

	t.Run("scenario: upload_and_read_back", func(t *testing.T) {
		cases := []struct {
			name    string
			payload []byte
		}{
			{name: "small_text", payload: []byte("hello ultrastructure")},
			{name: "json_payload", payload: []byte(`{"ok":true,"v":1}`)},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				caseID := uuid.NewString()
				objectKey := "objects/" + strings.ReplaceAll(tc.name, " ", "_") + "-" + caseID + ".txt"
				cacheKey := "object:" + caseID

				t.Cleanup(func() {
					_ = deps.redisClient.Del(ctx, cacheKey).Err()
					_, _ = deps.deleter.DeleteObject(ctx, &s3sdk.DeleteObjectInput{
						Bucket: aws.String(bucket),
						Key:    aws.String(objectKey),
					})
				})

				tx := deps.db.WithContext(ctx).Begin()
				if tx.Error != nil {
					t.Fatalf("begin tx: %v", tx.Error)
				}
				t.Cleanup(func() {
					_ = tx.Rollback().Error
				})

				// When
				if _, err := deps.uploader.PutObject(ctx, &s3sdk.PutObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(objectKey),
					Body:   bytes.NewReader(tc.payload),
				}); err != nil {
					t.Fatalf("put object: %v", err)
				}

				var id int64
				if err := tx.
					Raw(`INSERT INTO integration_objects (object_key, byte_size) VALUES (?, ?) RETURNING id`, objectKey, len(tc.payload)).
					Scan(&id).Error; err != nil {
					t.Fatalf("insert row: %v", err)
				}

				if err := deps.redisClient.Set(ctx, cacheKey, objectKey, 0).Err(); err != nil {
					t.Fatalf("redis set: %v", err)
				}

				// Then
				cachedKey, err := deps.redisClient.Get(ctx, cacheKey).Result()
				if err != nil {
					t.Fatalf("redis get: %v", err)
				}
				if cachedKey != objectKey {
					t.Fatalf("cached key mismatch: got=%q want=%q", cachedKey, objectKey)
				}

				var dbKey string
				var dbSize int
				if err := tx.
					Raw(`SELECT object_key, byte_size FROM integration_objects WHERE id = ?`, id).
					Row().
					Scan(&dbKey, &dbSize); err != nil {
					t.Fatalf("select row: %v", err)
				}
				if dbKey != objectKey || dbSize != len(tc.payload) {
					t.Fatalf("db row mismatch: got=(%q,%d) want=(%q,%d)", dbKey, dbSize, objectKey, len(tc.payload))
				}

				getOut, err := deps.downloader.GetObject(ctx, &s3sdk.GetObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(cachedKey),
				})
				if err != nil {
					t.Fatalf("get object: %v", err)
				}
				defer func() { _ = getOut.Body.Close() }()

				body, err := io.ReadAll(getOut.Body)
				if err != nil {
					t.Fatalf("read object body: %v", err)
				}
				if !bytes.Equal(body, tc.payload) {
					t.Fatalf("object payload mismatch: got=%q want=%q", string(body), string(tc.payload))
				}
			})
		}
	})

	t.Run("scenario: missing_object_returns_error", func(t *testing.T) {
		cases := []struct {
			name string
			key  string
		}{
			{name: "unknown_key_1", key: "missing/" + uuid.NewString()},
			{name: "unknown_key_2", key: "missing/" + uuid.NewString()},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := deps.downloader.GetObject(ctx, &s3sdk.GetObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(tc.key),
				})
				if err == nil {
					t.Fatalf("expected error for missing key %q", tc.key)
				}
			})
		}
	})
}

func startE2EApp(t *testing.T, pgURL string, redisCtr *RedisContainer, minio *MinIOContainer, bucket string) e2eDeps {
	t.Helper()

	var deps e2eDeps

	app := ditest.New(t,
		di.Diagnostics(),
		lifecycle.Module(),
		otel.Module(),
		database.Module(),
		rd.Module(rd.UseInterfaces()),
		uss3.Module(uss3.UseInterfaces()),

		di.Replace(database.Config{
			Dialect:    "postgres",
			Datasource: pgURL,
		}),
		di.Replace(rd.Config{
			Addr:     redisCtr.Addr(),
			Password: redisCtr.Password,
			Protocol: 3,
		}),
		di.Replace(uss3.Config{
			Region:          minio.Region(),
			Endpoint:        minio.Endpoint,
			AccessKeyID:     minio.AccessKey,
			SecretAccessKey: minio.SecretKey,
			UsePathStyle:    true,
			Bucket:          bucket,
		}),

		di.Populate(&deps.db),
		di.Populate(&deps.redisClient),
		di.Populate(&deps.bucketManager),
		di.Populate(&deps.uploader),
		di.Populate(&deps.downloader),
		di.Populate(&deps.deleter),
	)
	app.RequireStart()
	t.Cleanup(app.RequireStop)

	return deps
}

func mustCreateSchema(t *testing.T, ctx context.Context, db *gorm.DB) {
	t.Helper()
	if err := db.WithContext(ctx).Exec(`
		CREATE TABLE IF NOT EXISTS integration_objects (
			id BIGSERIAL PRIMARY KEY,
			object_key TEXT NOT NULL UNIQUE,
			byte_size INT NOT NULL
		);
	`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
}

func mustCreateBucket(t *testing.T, ctx context.Context, bucketManager uss3.BucketManager, bucket string) {
	t.Helper()
	if _, err := bucketManager.CreateBucket(ctx, &s3sdk.CreateBucketInput{
		Bucket: aws.String(bucket),
	}); err != nil {
		t.Fatalf("create bucket: %v", err)
	}
}
