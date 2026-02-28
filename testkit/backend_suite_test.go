package testkit

import "testing"

func TestWithBackendSuiteDefaults(t *testing.T) {
	opts := withBackendSuiteDefaults(BackendSuiteOptions{})

	if opts.MigrationDir != defaultBackendMigrationDir {
		t.Fatalf("unexpected migration dir: got=%q want=%q", opts.MigrationDir, defaultBackendMigrationDir)
	}
	if opts.MigrationDialect != defaultBackendMigrationDialect {
		t.Fatalf("unexpected migration dialect: got=%q want=%q", opts.MigrationDialect, defaultBackendMigrationDialect)
	}
	if opts.CasePrefix != defaultBackendCasePrefix {
		t.Fatalf("unexpected case prefix: got=%q want=%q", opts.CasePrefix, defaultBackendCasePrefix)
	}
}

func TestCleanIDFragment(t *testing.T) {
	got := cleanIDFragment("Scenario: Upload And Read/Back")
	if want := "scenario_upload_and_read_back"; got != want {
		t.Fatalf("unexpected clean id: got=%q want=%q", got, want)
	}
}

func TestMakePostgresDatabaseName(t *testing.T) {
	got := makePostgresDatabaseName("it", "Case#1")
	if want := "it_case_1"; got != want {
		t.Fatalf("unexpected postgres db name: got=%q want=%q", got, want)
	}
}

func TestMakeRedisPrefix(t *testing.T) {
	got := makeRedisPrefix("it", "case/a")
	if want := "it_case_a:"; got != want {
		t.Fatalf("unexpected redis prefix: got=%q want=%q", got, want)
	}
}

func TestMakeMinIOBucketName(t *testing.T) {
	got := makeMinIOBucketName("it", "Case_A")
	if want := "it-case-a"; got != want {
		t.Fatalf("unexpected bucket name: got=%q want=%q", got, want)
	}
}

func TestQuoteIdentifier(t *testing.T) {
	got := quoteIdentifier(`demo"name`)
	if want := `"demo""name"`; got != want {
		t.Fatalf("unexpected quoted identifier: got=%q want=%q", got, want)
	}
}

func TestBackendCaseConfigsAndReplaces(t *testing.T) {
	c := &BackendCase{
		PostgresURL: "postgres://user:pass@127.0.0.1:5432/db?sslmode=disable",
		RedisPrefix: "it_case:",
		MinIOBucket: "it-case",
		Redis: &RedisContainer{
			Host:     "127.0.0.1",
			Port:     "6379",
			Password: "secret",
		},
		MinIO: &MinIOContainer{
			Endpoint:  "http://127.0.0.1:9000",
			AccessKey: "minio",
			SecretKey: "miniopass",
		},
	}

	dbCfg := c.DatabaseConfig()
	if dbCfg.Driver != "postgres" || dbCfg.Datasource != c.PostgresURL {
		t.Fatalf("unexpected db config: %+v", dbCfg)
	}

	redisCfg := c.RedisConfig()
	if redisCfg.Addr != "127.0.0.1:6379" || redisCfg.Password != "secret" {
		t.Fatalf("unexpected redis config: %+v", redisCfg)
	}

	s3Cfg := c.MinIOConfig()
	if s3Cfg.Bucket != "it-case" || s3Cfg.Endpoint != "http://127.0.0.1:9000" {
		t.Fatalf("unexpected minio config: %+v", s3Cfg)
	}

	if got := c.RedisKey("k"); got != "it_case:k" {
		t.Fatalf("unexpected redis key: got=%q", got)
	}

	replaces := c.Replaces()
	if len(replaces) != 3 {
		t.Fatalf("unexpected replaces length: got=%d want=3", len(replaces))
	}
}
