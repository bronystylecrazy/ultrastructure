package testkit

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

var insertIntoTablePattern = regexp.MustCompile(`(?is)\binsert\s+into\s+((?:"[^"]+"|[a-zA-Z_][a-zA-Z0-9_]*)(?:\.(?:"[^"]+"|[a-zA-Z_][a-zA-Z0-9_]*))?)`)

// Seeder executes SQL seeds against a BackendCase database.
type Seeder struct {
	t  testing.TB
	db *sql.DB
}

// SeededData represents an applied seed and supports cleanup with Reset.
type SeededData struct {
	t       testing.TB
	db      *sql.DB
	downSQL string
	tables  []string

	resetOnce sync.Once
}

// Seeder opens a SQL seeder bound to this case database.
func (c *BackendCase) Seeder(t testing.TB) *Seeder {
	t.Helper()

	db, err := sql.Open("postgres", c.PostgresURL)
	if err != nil {
		t.Fatalf("open postgres for seeder: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return &Seeder{t: t, db: db}
}

// Seed applies upSQL and returns a handle that can be reset with defer.
// Optional downSQL has priority during Reset.
func (s *Seeder) Seed(upSQL string, downSQL ...string) *SeededData {
	s.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := s.exec(ctx, upSQL); err != nil {
		s.t.Fatalf("seed up sql: %v", err)
	}

	var down string
	if len(downSQL) > 0 {
		down = downSQL[0]
	}

	return &SeededData{
		t:       s.t,
		db:      s.db,
		downSQL: strings.TrimSpace(down),
		tables:  insertTablesFromSQL(upSQL),
	}
}

// Reset removes seeded data using downSQL when provided; otherwise it truncates inserted tables.
func (d *SeededData) Reset() {
	d.t.Helper()

	d.resetOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if d.downSQL != "" {
			if err := d.exec(ctx, d.downSQL); err != nil {
				d.t.Fatalf("seed reset sql: %v", err)
			}
			return
		}

		if len(d.tables) == 0 {
			return
		}

		stmt := "TRUNCATE TABLE " + strings.Join(uniqueStrings(d.tables), ", ") + " RESTART IDENTITY CASCADE"
		if err := d.exec(ctx, stmt); err != nil {
			d.t.Fatalf("seed auto reset: %v", err)
		}
	})
}

func (s *Seeder) exec(ctx context.Context, query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, query)
	return err
}

func (d *SeededData) exec(ctx context.Context, query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	_, err := d.db.ExecContext(ctx, query)
	return err
}

func insertTablesFromSQL(sqlText string) []string {
	matches := insertIntoTablePattern.FindAllStringSubmatch(sqlText, -1)
	if len(matches) == 0 {
		return nil
	}

	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		table := strings.TrimSpace(m[1])
		if table == "" {
			continue
		}
		out = append(out, table)
	}
	return uniqueStrings(out)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
