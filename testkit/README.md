# testkit

`testkit` is a small helper package for integration tests that use [testcontainers-go](https://github.com/testcontainers/testcontainers-go).

## Features

- `NewSuite(t)`:
  - checks Docker availability,
  - provides a shared context,
  - auto-terminates containers on test cleanup.
- `RequireIntegration(t)`:
  - optional env guard (`RUN_INTEGRATION_TESTS=1`).
- `StartEMQX(opts)`:
  - starts EMQX with dashboard credentials and returns connection metadata.
- `StartPostgres(opts)`:
  - starts PostgreSQL and returns `postgres://` URL via `pg.URL()`.
- `StartRedis(opts)`:
  - starts Redis and returns host/port via `rd.Addr()`.
- `StartMinIO(opts)`:
  - starts MinIO and returns API/console endpoints.
- `NewBackendSuite(t, opts)`:
  - starts Postgres/Redis/MinIO once per parent test,
  - provisions isolated resources per case with `suite.NewCase(...)`:
    - Postgres: unique database + migration run,
    - Redis: unique key namespace prefix,
    - MinIO: unique bucket.
- `NewBackend(t, migrationFS, migrationDir...)`:
  - minimal constructor for backend integration suites.
- `tc.Seeder(t)`:
  - apply inline SQL seed and clean it with `defer seed.Reset()`.

## Usage

### 1) Direct usage in an integration test

```go
//go:build integration

import (
	"embed"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/testkit"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/migrations/*.sql
var migrations embed.FS

func TestMyIntegration(t *testing.T) {
	require := require.New(t)

	testkit.RequireIntegration(t)

	suite := testkit.NewBackend(t, migrations, "testdata/migrations")

	t.Run("case_a", func(t *testing.T) {
		t.Parallel()
		tc := suite.Case(t)

		require.NotEmpty(tc.PostgresURL)
		require.NotEmpty(tc.RedisPrefix)
		require.NotEmpty(tc.MinIOBucket)
	})
}
```

Minimal DI wiring:

```go
tc := suite.Case(t)
nodes := append(tc.Replaces(),
	di.Populate(&svcA),
	di.Populate(&svcB),
)
app := ustest.New(t, nodes...).RequireStart()
t.Cleanup(app.RequireStop)
```

Inline seed with reset:

```go
tc := suite.Case(t)
seeder := tc.Seeder(t)

seed := seeder.Seed(`
	INSERT INTO customers (id, name) VALUES
		('cust-1', 'Alice'),
		('cust-2', 'Bob')
	ON CONFLICT (id) DO NOTHING;
`)
defer seed.Reset()
```

Custom reset SQL:

```go
seed := seeder.Seed(upSQL, `
	DELETE FROM products WHERE id IN ('prod-1');
	DELETE FROM customers WHERE id IN ('cust-1', 'cust-2');
`)
defer seed.Reset()
```

### 2) API-level integration test (recommended for backend domains)

```go
//go:build integration

import (
	"bytes"
	"embed"
	"net/http/httptest"
	"testing"

	redis "github.com/redis/go-redis/v9"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/testkit"
	"github.com/bronystylecrazy/ultrastructure/ustest"
	"github.com/gofiber/fiber/v3"

	"yourrepo/app"
)

//go:embed testdata/migrations/*.sql
var migrations embed.FS

func TestOrderAPI(t *testing.T) {
	testkit.RequireIntegration(t)
	suite := testkit.NewBackend(t, migrations, "testdata/migrations") // containers once

	t.Run("create_order", func(t *testing.T) {
		t.Parallel()
		tc := suite.Case(t) // isolated postgres db + redis namespace + minio bucket

		seed := tc.Seeder(t).Seed(`
			INSERT INTO customers (id, name) VALUES ('cust-1', 'Alice')
			ON CONFLICT (id) DO NOTHING;
		`)
		defer seed.Reset()

		rdb := redis.NewClient(&redis.Options{
			Addr:     tc.Redis.Addr(),
			Password: tc.Redis.Password,
			Protocol: 3,
		})
		t.Cleanup(func() { _ = rdb.Close() })
		_ = rdb.Set(t.Context(), tc.RedisKey("feature:checkout"), "on", 0).Err()

		var api *fiber.App
		nodes := append(tc.Replaces(),
			app.Module(),
			di.Populate(&api),
		)
		testApp := ustest.New(t, nodes...).RequireStart()
		t.Cleanup(testApp.RequireStop)

		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(`{"customer_id":"cust-1","amount":1000}`))
		req.Header.Set("Content-Type", "application/json")

		res, err := api.Test(req)
		if err != nil {
			t.Fatalf("api test request: %v", err)
		}
		if res.StatusCode != 201 {
			t.Fatalf("status code mismatch: got=%d want=%d", res.StatusCode, 201)
		}
	})
}
```

### 3) Consumer repo pattern (`di.App` + `Populate`)

```go
//go:build integration

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"yourrepo/domain/order"
	"yourrepo/domain/payment"
)

func TestCheckout(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	testkit.RequireIntegration(t)

	var orderSvc *order.Service
	var paymentSvc *payment.Service

	NewSuiteEnv(t).
		Populate(&orderSvc, &paymentSvc).
		Start()

	require.NotNil(orderSvc)
	require.NotNil(paymentSvc)

	// perform assertions with orderSvc / paymentSvc
	assert.True(true)
}
```

Example `SuiteEnv` API shape:
- `NewSuiteEnv(t)` starts required containers once for the test.
- `Populate(&svcA, &svcB)` appends `di.Populate(...)` targets.
- `Start()` builds and starts `di.App(...)`, and registers cleanup.

```go
// test/integration/testkit/suite_env.go
package testkit

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/x/redis"
	ustk "github.com/bronystylecrazy/ultrastructure/testkit"
	"go.uber.org/fx/fxtest"

	"yourrepo/app"
)

type SuiteEnv struct {
	t     *testing.T
	nodes []any
	app   *fxtest.App
}

func NewSuiteEnv(t *testing.T) *SuiteEnv {
	t.Helper()

	suite := ustk.NewSuite(t)
	pg := suite.StartPostgres(ustk.PostgresOptions{})
	redisCtr := suite.StartRedis(ustk.RedisOptions{})

	return &SuiteEnv{
		t: t,
		nodes: []any{
			app.Module(),
			di.Replace(database.Config{
				Driver:     "postgres",
				Datasource: pg.URL(),
			}),
			di.Replace(rd.Config{
				Addr:     redisCtr.Addr(),
				Password: redisCtr.Password,
				Protocol: 3,
			}),
		},
	}
}

func (e *SuiteEnv) Populate(targets ...any) *SuiteEnv {
	e.t.Helper()
	for _, target := range targets {
		e.nodes = append(e.nodes, di.Populate(target))
	}
	return e
}

func (e *SuiteEnv) Start() *SuiteEnv {
	e.t.Helper()
	e.app = fxtest.New(e.t, di.App(e.nodes...).Build())
	e.app.RequireStart()
	e.t.Cleanup(e.app.RequireStop)
	return e
}
```

### 4) Scenario/case style (parallel-safe)

- Use `t.Run("scenario: ...")` for business flows.
- Use table-driven cases inside each scenario.
- In each case: `tc := tc`, `t.Parallel()`, then `env := suite.Case(t)`.
- Isolate with:
  - Postgres: unique DB + migrations,
  - Redis: `env.RedisKey("...")`,
  - MinIO: case bucket via `env.MinIOBucket`.

`testify` pattern recommendation:
- use `require.*` for setup and critical preconditions.
- use `assert.*` for post-action validations where you want multiple checks.

## Suggested Project Structure (Consumer Repo)

```text
yourrepo/
  app/
  domain/
    order/
      order_integration_test.go
    payment/
      payment_integration_test.go
  test/
    integration/
      testkit/
        suite_env.go
        fixtures.go
        kafka.go
      scenarios/
        checkout_flow_test.go
        refund_flow_test.go
      seeds/
        sql/
          base_schema.sql
          seed_users.sql
      helpers/
        assert.go
        http.go
```

Recommended split:
- Domain-local integration tests:
  - `domain/*/*_integration_test.go`
- Shared integration bootstrapping/helpers:
  - `test/integration/testkit/*`
- Cross-domain end-to-end scenarios:
  - `test/integration/scenarios/*`
- Seed data and helper assertions:
  - `test/integration/seeds/*`, `test/integration/helpers/*`

## Run

- Unit tests only:
  - `go test ./testkit`
- Integration tests (requires Docker):
  - `RUN_INTEGRATION_TESTS=1 go test -tags integration ./testkit -v`
- Consumer repo integration tests:
  - `RUN_INTEGRATION_TESTS=1 go test -tags integration ./... -v`
