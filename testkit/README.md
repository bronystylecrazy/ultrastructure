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

## Usage

### 1) Direct usage in an integration test

```go
//go:build integration

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/testkit"
	"github.com/stretchr/testify/require"
)

func TestMyIntegration(t *testing.T) {
	require := require.New(t)

	testkit.RequireIntegration(t)

	suite := testkit.NewSuite(t)
	pg := suite.StartPostgres(testkit.PostgresOptions{})
	rd := suite.StartRedis(testkit.RedisOptions{})
	s3 := suite.StartMinIO(testkit.MinIOOptions{})

	require.NotEmpty(pg.URL())
	require.NotEmpty(rd.Addr())
	require.NotEmpty(s3.Endpoint)
}
```

### 2) Consumer repo pattern (`di.App` + `Populate`)

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

	"github.com/bronystylecrazy/ultrastructure/caching/rd"
	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
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
				Dialect:    "postgres",
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

### 3) Scenario/case style (parallel-safe)

- Use `t.Run("scenario: ...")` for business flows.
- Use table-driven cases inside each scenario.
- In each case: `tc := tc`, `t.Parallel()`, unique `caseID`.
- Isolate with transaction rollback + unique Redis/S3 prefixes.

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
