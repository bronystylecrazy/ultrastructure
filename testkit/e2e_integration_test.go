//go:build integration

package testkit

import (
	"bytes"
	"embed"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bronystylecrazy/ultrastructure/di"
	uss3 "github.com/bronystylecrazy/ultrastructure/storage/s3"
	"github.com/bronystylecrazy/ultrastructure/ustest"
	rd "github.com/bronystylecrazy/ultrastructure/x/redis"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

//go:embed testdata/migrations/*.sql
var e2eMigrations embed.FS

type e2eDeps struct {
	db          *gorm.DB
	redisClient rd.RedisClient
	uploader    uss3.Uploader
	downloader  uss3.Downloader
	deleter     uss3.Deleter
}

func TestE2EScenarios_PostgresRedisMinIO(t *testing.T) {
	RequireIntegration(t)

	suite := NewBackend(t, e2eMigrations, "testdata/migrations")

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
				tcEnv := suite.Case(t)
				ctx := t.Context()
				deps := startE2EApp(t, tcEnv)

				caseID := uuid.NewString()
				objectKey := "objects/" + tc.name + "-" + caseID + ".txt"
				cacheKey := tcEnv.RedisKey("object:" + caseID)

				t.Cleanup(func() {
					_, _ = deps.deleter.DeleteObject(ctx, &s3sdk.DeleteObjectInput{
						Bucket: aws.String(tcEnv.MinIOBucket),
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
					Bucket: aws.String(tcEnv.MinIOBucket),
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
					Bucket: aws.String(tcEnv.MinIOBucket),
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
				tcEnv := suite.Case(t)
				ctx := t.Context()
				deps := startE2EApp(t, tcEnv)

				_, err := deps.downloader.GetObject(ctx, &s3sdk.GetObjectInput{
					Bucket: aws.String(tcEnv.MinIOBucket),
					Key:    aws.String(tc.key),
				})
				if err == nil {
					t.Fatalf("expected error for missing key %q", tc.key)
				}
			})
		}
	})
}

func startE2EApp(t *testing.T, tc *BackendCase) e2eDeps {
	t.Helper()

	var deps e2eDeps
	nodes := append(tc.Replaces(),
		di.Populate(&deps.db),
		di.Populate(&deps.redisClient),
		di.Populate(&deps.uploader),
		di.Populate(&deps.downloader),
		di.Populate(&deps.deleter),
	)

	app := ustest.New(t, nodes...)
	app.RequireStart()
	t.Cleanup(app.RequireStop)

	return deps
}
