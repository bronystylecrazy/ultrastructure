package testkit

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	uss3 "github.com/bronystylecrazy/ultrastructure/storage/s3"
	rd "github.com/bronystylecrazy/ultrastructure/x/redis"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	redis "github.com/redis/go-redis/v9"
)

const (
	defaultBackendMigrationDir     = "migrations"
	defaultBackendMigrationDialect = "postgres"
	defaultBackendCasePrefix       = "it"
)

// BackendSuiteOptions controls shared container setup and per-case migration behavior.
type BackendSuiteOptions struct {
	Postgres         PostgresOptions
	Redis            RedisOptions
	MinIO            MinIOOptions
	MigrationFS      fs.FS
	MigrationDir     string
	MigrationDialect string
	CasePrefix       string
}

// BackendCaseOptions customizes isolated resources for a single test case.
type BackendCaseOptions struct {
	Name             string
	PostgresDatabase string
	RedisPrefix      string
	MinIOBucket      string
	MigrationFS      fs.FS
	MigrationDir     string
	SkipMigrations   bool
}

// BackendSuite starts shared backend dependencies once and provisions isolated resources per test case.
type BackendSuite struct {
	suite   *Suite
	pg      *PostgresContainer
	rd      *RedisContainer
	minio   *MinIOContainer
	redis   *redis.Client
	s3      *s3sdk.Client
	migFS   fs.FS
	migDir  string
	dialect string
	prefix  string

	migrationsMu sync.Mutex
}

// BackendCase is an isolated test case environment.
type BackendCase struct {
	ID               string
	PostgresURL      string
	PostgresDatabase string
	RedisPrefix      string
	MinIOBucket      string

	Postgres *PostgresContainer
	Redis    *RedisContainer
	MinIO    *MinIOContainer
}

// RedisKey returns a case-scoped Redis key.
func (c *BackendCase) RedisKey(key string) string {
	return c.RedisPrefix + key
}

// DatabaseConfig returns a prefilled postgres database.Config for this case.
func (c *BackendCase) DatabaseConfig() database.Config {
	return database.Config{
		Driver:     "postgres",
		Datasource: c.PostgresURL,
	}
}

// RedisConfig returns a prefilled redis config for this case.
func (c *BackendCase) RedisConfig() rd.Config {
	return rd.Config{
		Addr:     c.Redis.Addr(),
		Password: c.Redis.Password,
		Protocol: 3,
	}
}

// MinIOConfig returns a prefilled S3 config for this case bucket.
func (c *BackendCase) MinIOConfig() uss3.Config {
	return uss3.Config{
		Region:          c.MinIO.Region(),
		Endpoint:        c.MinIO.Endpoint,
		AccessKeyID:     c.MinIO.AccessKey,
		SecretAccessKey: c.MinIO.SecretKey,
		UsePathStyle:    true,
		Bucket:          c.MinIOBucket,
	}
}

// Replaces returns DI replacement nodes for DB, Redis, and MinIO.
func (c *BackendCase) Replaces() []any {
	return []any{
		di.Replace(c.DatabaseConfig()),
		di.Replace(c.RedisConfig()),
		di.Replace(c.MinIOConfig()),
	}
}

// NewBackend is the minimal constructor with defaults; customize via NewBackendSuite when needed.
func NewBackend(t testing.TB, migrationFS fs.FS, migrationDir ...string) *BackendSuite {
	t.Helper()
	dir := ""
	if len(migrationDir) > 0 {
		dir = migrationDir[0]
	}
	return NewBackendSuite(t, BackendSuiteOptions{
		MigrationFS:  migrationFS,
		MigrationDir: dir,
	})
}

// NewBackendSuite starts shared Postgres/Redis/MinIO containers and prepares clients for per-case provisioning.
func NewBackendSuite(t testing.TB, opts BackendSuiteOptions) *BackendSuite {
	t.Helper()

	opts = withBackendSuiteDefaults(opts)
	suite := NewSuite(t)

	pg := suite.StartPostgres(opts.Postgres)
	rd := suite.StartRedis(opts.Redis)
	minio := suite.StartMinIO(opts.MinIO)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     rd.Addr(),
		Password: rd.Password,
		Protocol: 3,
	})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	awsCfg, err := config.LoadDefaultConfig(
		suite.Context(),
		config.WithRegion(minio.Region()),
		config.WithBaseEndpoint(minio.Endpoint),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(minio.AccessKey, minio.SecretKey, "")),
	)
	if err != nil {
		t.Fatalf("load aws config for minio: %v", err)
	}
	s3Client := s3sdk.NewFromConfig(awsCfg, func(o *s3sdk.Options) {
		o.UsePathStyle = true
	})

	return &BackendSuite{
		suite:   suite,
		pg:      pg,
		rd:      rd,
		minio:   minio,
		redis:   redisClient,
		s3:      s3Client,
		migFS:   opts.MigrationFS,
		migDir:  opts.MigrationDir,
		dialect: opts.MigrationDialect,
		prefix:  opts.CasePrefix,
	}
}

// NewCase provisions an isolated Postgres DB, Redis namespace, and MinIO bucket for a test case.
func (s *BackendSuite) NewCase(t testing.TB, opts BackendCaseOptions) *BackendCase {
	t.Helper()

	opts = s.withBackendCaseDefaults(t, opts)
	ctx := s.suite.Context()

	if err := s.createDatabase(ctx, opts.PostgresDatabase); err != nil {
		t.Fatalf("create postgres database %q: %v", opts.PostgresDatabase, err)
	}
	t.Cleanup(func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.dropDatabase(dropCtx, opts.PostgresDatabase); err != nil {
			t.Logf("cleanup drop postgres database %q: %v", opts.PostgresDatabase, err)
		}
	})

	pgURL := s.pg.URLForDatabase(opts.PostgresDatabase)
	if !opts.SkipMigrations {
		migFS := opts.MigrationFS
		if migFS == nil {
			migFS = s.migFS
		}
		migDir := opts.MigrationDir
		if migDir == "" {
			migDir = s.migDir
		}
		if err := s.runMigrations(ctx, pgURL, migFS, migDir); err != nil {
			t.Fatalf("run postgres migrations for database %q: %v", opts.PostgresDatabase, err)
		}
	}

	if _, err := s.s3.CreateBucket(ctx, &s3sdk.CreateBucketInput{Bucket: &opts.MinIOBucket}); err != nil {
		t.Fatalf("create minio bucket %q: %v", opts.MinIOBucket, err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.deleteBucketRecursive(cleanupCtx, opts.MinIOBucket); err != nil {
			t.Logf("cleanup minio bucket %q: %v", opts.MinIOBucket, err)
		}
	})

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.deleteRedisNamespace(cleanupCtx, opts.RedisPrefix); err != nil {
			t.Logf("cleanup redis namespace %q: %v", opts.RedisPrefix, err)
		}
	})

	return &BackendCase{
		ID:               caseIDFromDatabase(opts.PostgresDatabase),
		PostgresURL:      pgURL,
		PostgresDatabase: opts.PostgresDatabase,
		RedisPrefix:      opts.RedisPrefix,
		MinIOBucket:      opts.MinIOBucket,
		Postgres:         s.pg,
		Redis:            s.rd,
		MinIO:            s.minio,
	}
}

// Case creates an isolated test case using defaults derived from t.Name().
func (s *BackendSuite) Case(t testing.TB) *BackendCase {
	t.Helper()
	return s.NewCase(t, BackendCaseOptions{})
}

func (s *BackendSuite) withBackendCaseDefaults(t testing.TB, opts BackendCaseOptions) BackendCaseOptions {
	caseID := cleanIDFragment(opts.Name)
	if caseID == "" {
		caseID = cleanIDFragment(t.Name())
	}
	if caseID == "" {
		caseID = "case"
	}
	caseID = caseID + "_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")

	if opts.PostgresDatabase == "" {
		opts.PostgresDatabase = makePostgresDatabaseName(s.prefix, caseID)
	}
	if opts.RedisPrefix == "" {
		opts.RedisPrefix = makeRedisPrefix(s.prefix, caseID)
	}
	if opts.MinIOBucket == "" {
		opts.MinIOBucket = makeMinIOBucketName(s.prefix, caseID)
	}
	return opts
}

func (s *BackendSuite) createDatabase(ctx context.Context, database string) error {
	adminDB, err := sql.Open("postgres", s.pg.URL())
	if err != nil {
		return err
	}
	defer func() { _ = adminDB.Close() }()

	if err := adminDB.PingContext(ctx); err != nil {
		return err
	}
	_, err = adminDB.ExecContext(ctx, `CREATE DATABASE `+quoteIdentifier(database))
	return err
}

func (s *BackendSuite) dropDatabase(ctx context.Context, database string) error {
	adminDB, err := sql.Open("postgres", s.pg.URL())
	if err != nil {
		return err
	}
	defer func() { _ = adminDB.Close() }()

	if err := adminDB.PingContext(ctx); err != nil {
		return err
	}
	if _, err := adminDB.ExecContext(ctx, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, database); err != nil {
		return err
	}
	_, err = adminDB.ExecContext(ctx, `DROP DATABASE IF EXISTS `+quoteIdentifier(database))
	return err
}

func (s *BackendSuite) runMigrations(ctx context.Context, postgresURL string, migrationFS fs.FS, migrationDir string) error {
	if migrationDir == "" {
		migrationDir = defaultBackendMigrationDir
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	s.migrationsMu.Lock()
	defer s.migrationsMu.Unlock()

	if err := goose.SetDialect(s.dialect); err != nil {
		return err
	}

	path := migrationDir
	if migrationFS != nil {
		base, subErr := fs.Sub(migrationFS, migrationDir)
		if subErr != nil {
			return subErr
		}
		goose.SetBaseFS(base)
		path = "."
	} else {
		goose.SetBaseFS(nil)
	}

	err = goose.UpContext(ctx, db, path)
	if err != nil && !errors.Is(err, goose.ErrNoNextVersion) {
		return err
	}
	return nil
}

func (s *BackendSuite) deleteRedisNamespace(ctx context.Context, prefix string) error {
	var cursor uint64
	for {
		keys, next, err := s.redis.Scan(ctx, cursor, prefix+"*", 200).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := s.redis.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

func (s *BackendSuite) deleteBucketRecursive(ctx context.Context, bucket string) error {
	var token *string
	for {
		out, err := s.s3.ListObjectsV2(ctx, &s3sdk.ListObjectsV2Input{
			Bucket:            &bucket,
			ContinuationToken: token,
		})
		if err != nil {
			return err
		}

		if len(out.Contents) > 0 {
			objects := make([]s3types.ObjectIdentifier, 0, len(out.Contents))
			for _, obj := range out.Contents {
				if obj.Key == nil {
					continue
				}
				objects = append(objects, s3types.ObjectIdentifier{Key: obj.Key})
			}
			if len(objects) > 0 {
				if _, err := s.s3.DeleteObjects(ctx, &s3sdk.DeleteObjectsInput{
					Bucket: &bucket,
					Delete: &s3types.Delete{Objects: objects, Quiet: aws.Bool(true)},
				}); err != nil {
					return err
				}
			}
		}

		if !aws.ToBool(out.IsTruncated) {
			break
		}
		token = out.NextContinuationToken
	}

	_, err := s.s3.DeleteBucket(ctx, &s3sdk.DeleteBucketInput{Bucket: &bucket})
	return err
}

func withBackendSuiteDefaults(opts BackendSuiteOptions) BackendSuiteOptions {
	if opts.MigrationDir == "" {
		opts.MigrationDir = defaultBackendMigrationDir
	}
	if opts.MigrationDialect == "" {
		opts.MigrationDialect = defaultBackendMigrationDialect
	}
	if opts.CasePrefix == "" {
		opts.CasePrefix = defaultBackendCasePrefix
	}
	return opts
}

func caseIDFromDatabase(database string) string {
	if idx := strings.LastIndex(database, "_"); idx > 0 && idx+1 < len(database) {
		return database[idx+1:]
	}
	return database
}

func makePostgresDatabaseName(prefix, fragment string) string {
	base := cleanIDFragment(prefix + "_" + fragment)
	if base == "" {
		base = "it_case"
	}
	if base[0] >= '0' && base[0] <= '9' {
		base = "db_" + base
	}
	if len(base) > 63 {
		base = base[:63]
	}
	return base
}

func makeRedisPrefix(prefix, fragment string) string {
	base := cleanIDFragment(prefix + "_" + fragment)
	if base == "" {
		base = "it_case"
	}
	return base + ":"
}

func makeMinIOBucketName(prefix, fragment string) string {
	base := cleanIDFragment(prefix + "-" + fragment)
	base = strings.ReplaceAll(base, "_", "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "it-case"
	}
	if len(base) < 3 {
		base = base + strings.Repeat("a", 3-len(base))
	}
	if len(base) > 63 {
		base = base[:63]
	}
	return base
}

func cleanIDFragment(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range raw {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	clean := strings.Trim(b.String(), "_")
	return clean
}

func quoteIdentifier(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}
