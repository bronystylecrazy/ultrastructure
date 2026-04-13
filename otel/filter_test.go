package otel

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestParseDebugAllowlist(t *testing.T) {
	rules := parseDebugAllowlist([]string{
		"layer:web.api",
		"logger:sqlc",
		"file:web/fiber.go:120-180",
		"func:(*Websocket).Init:40-75",
		"function:pkg/service.Run:9",
	})

	if !layerAllowed("web.api.v1", rules.layers) {
		t.Fatalf("expected layer rule to match prefix")
	}
	if !loggerAllowed("sqlc.query", rules.loggers) {
		t.Fatalf("expected logger rule to match prefix")
	}
	if !locationAllowed("/workspace/web/fiber.go", 150, rules.files, fileValueMatches) {
		t.Fatalf("expected file rule with range to match")
	}
	if locationAllowed("/workspace/web/fiber.go", 181, rules.files, fileValueMatches) {
		t.Fatalf("did not expect file rule to match outside range")
	}
	if !locationAllowed("github.com/acme/realtime.(*Websocket).Init", 50, rules.functions, functionValueMatches) {
		t.Fatalf("expected function rule with range to match")
	}
	if !locationAllowed("pkg/service.Run", 9, rules.functions, functionValueMatches) {
		t.Fatalf("expected single-line function rule to match")
	}
}

func TestFilterDebugCoreFiltersByLoggerLayerFileAndFunction(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(FilterDebugCore(core, parseDebugAllowlist([]string{
		"layer:web.api",
		"logger:sqlc",
		"file:web/fiber.go:120-180",
		"func:(*Websocket).Init:40-75",
	})))

	logger.Named("sqlc").Debug("keep logger")
	logger.Named("noise").With(appLayerField("web.api.v1")).Debug("keep layer")
	writeEntry(t, logger.Named("noise"), zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "keep file range",
		Caller:  zapcore.EntryCaller{Defined: true, File: "/workspace/web/fiber.go", Line: 150},
	})
	writeEntry(t, logger.Named("noise"), zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "keep function range",
		Caller:  zapcore.EntryCaller{Defined: true, Function: "github.com/acme/realtime.(*Websocket).Init", Line: 60},
	})
	writeEntry(t, logger.Named("noise"), zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "drop unrelated",
		Caller:  zapcore.EntryCaller{Defined: true, File: "/workspace/noise.go", Line: 10, Function: "pkg/noise.Run"},
	})
	writeEntry(t, logger.Named("noise"), zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "drop unrelated info",
		Caller:  zapcore.EntryCaller{Defined: true, File: "/workspace/noise.go", Line: 10, Function: "pkg/noise.Run"},
	})
	writeEntry(t, logger.Named("noise"), zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "keep layer info",
		Caller:  zapcore.EntryCaller{Defined: true, File: "/workspace/noise.go", Line: 10, Function: "pkg/noise.Run"},
	}, appLayerField("web.api.v1"))
	writeEntry(t, logger.Named("noise"), zapcore.Entry{
		Level:   zapcore.WarnLevel,
		Message: "keep file warn",
		Caller:  zapcore.EntryCaller{Defined: true, File: "/workspace/web/fiber.go", Line: 150},
	})
	writeEntry(t, logger.Named("noise"), zapcore.Entry{
		Level:   zapcore.ErrorLevel,
		Message: "keep function error",
		Caller:  zapcore.EntryCaller{Defined: true, Function: "github.com/acme/realtime.(*Websocket).Init", Line: 60},
	})
	writeEntry(t, logger.Named("noise"), zapcore.Entry{
		Level:   zapcore.WarnLevel,
		Message: "drop unrelated warn",
		Caller:  zapcore.EntryCaller{Defined: true, File: "/workspace/noise.go", Line: 10, Function: "pkg/noise.Run"},
	})

	entries := logs.AllUntimed()
	if len(entries) != 7 {
		t.Fatalf("unexpected entry count: got %d want %d", len(entries), 7)
	}

	wantMessages := []string{
		"keep logger",
		"keep layer",
		"keep file range",
		"keep function range",
		"keep layer info",
		"keep file warn",
		"keep function error",
	}
	for i, want := range wantMessages {
		if entries[i].Message != want {
			t.Fatalf("unexpected message at %d: got %q want %q", i, entries[i].Message, want)
		}
	}
}

func TestFileRuleLineRange(t *testing.T) {
	rules := parseDebugAllowlist([]string{"file:web/fiber.go:120-180"})
	if !locationAllowed("/workspace/web/fiber.go", 120, rules.files, fileValueMatches) {
		t.Fatalf("expected range start to match")
	}
	if !locationAllowed("/workspace/web/fiber.go", 180, rules.files, fileValueMatches) {
		t.Fatalf("expected range end to match")
	}
	if locationAllowed("/workspace/web/fiber.go", 181, rules.files, fileValueMatches) {
		t.Fatalf("did not expect line after range to match")
	}
}

func TestFunctionRuleLineRange(t *testing.T) {
	rules := parseDebugAllowlist([]string{"func:(*Websocket).Init:40-75", "func:pkg/service.Run:9"})
	if !locationAllowed("github.com/acme/realtime.(*Websocket).Init", 40, rules.functions, functionValueMatches) {
		t.Fatalf("expected function range start to match")
	}
	if !locationAllowed("github.com/acme/realtime.(*Websocket).Init", 75, rules.functions, functionValueMatches) {
		t.Fatalf("expected function range end to match")
	}
	if locationAllowed("github.com/acme/realtime.(*Websocket).Init", 76, rules.functions, functionValueMatches) {
		t.Fatalf("did not expect line after function range to match")
	}
	if !locationAllowed("pkg/service.Run", 9, rules.functions, functionValueMatches) {
		t.Fatalf("expected single-line function rule to match")
	}
}

func TestSplitLocationRuleTreatsNonNumericSuffixAsPartOfValue(t *testing.T) {
	base, lines, ok := splitLocationRule("github.com/acme/pkg.(*Thing).Run")
	if !ok {
		t.Fatalf("expected function value without range to parse")
	}
	if base != "github.com/acme/pkg.(*Thing).Run" {
		t.Fatalf("unexpected base: got %q", base)
	}
	if lines.enabled() {
		t.Fatalf("did not expect line range to be enabled")
	}
}

func TestFilterDebugLoggerCoreFiltersDebugByLoggerName(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(FilterDebugLoggerCore(core, "sqlc", "web.api"))

	logger.Named("sqlc").Debug("keep exact")
	logger.Named("sqlc.query").Debug("keep child")
	logger.Named("gorm").Debug("drop other")
	logger.Debug("drop root")
	logger.Named("gorm").Info("keep info")
	logger.Named("gorm").Warn("drop warn")

	entries := logs.AllUntimed()
	if len(entries) != 2 {
		t.Fatalf("unexpected entry count: got %d want %d", len(entries), 2)
	}
}

func writeEntry(t *testing.T, logger *zap.Logger, entry zapcore.Entry, fields ...zap.Field) {
	t.Helper()
	zapFields := make([]zapcore.Field, len(fields))
	copy(zapFields, fields)
	checked := logger.Core().Check(entry, nil)
	if checked == nil {
		return
	}
	checked.Write(zapFields...)
}
