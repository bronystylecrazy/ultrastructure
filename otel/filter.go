package otel

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const appLayerFieldKey = "app.layer"

// FilterFieldsCore drops fields with matching keys before writing to the core.
func FilterFieldsCore(core zapcore.Core, dropKeys ...string) zapcore.Core {
	return filterFieldsCore{
		Core: core,
		drop: makeDropSet(dropKeys),
	}
}

// FilterDebugLoggerCore keeps entries only for allowed logger names.
func FilterDebugLoggerCore(core zapcore.Core, allowNames ...string) zapcore.Core {
	rules := parseDebugAllowlist(prefixEntries("logger", allowNames))
	return FilterDebugCore(core, rules)
}

// FilterDebugCore keeps entries only when they match at least one allowlist rule.
func FilterDebugCore(core zapcore.Core, rules debugAllowlist) zapcore.Core {
	return debugFilterCore{
		Core:  core,
		rules: rules,
	}
}

type filterFieldsCore struct {
	zapcore.Core
	drop map[string]struct{}
}

type debugFilterCore struct {
	zapcore.Core
	rules debugAllowlist
	layer string
}

type debugAllowlist struct {
	layers    map[string]struct{}
	loggers   map[string]struct{}
	files     []debugLocationRule
	functions []debugLocationRule
}

type debugLocationRule struct {
	value string
	lines lineRange
}

type lineRange struct {
	start int
	end   int
}

func (r lineRange) enabled() bool {
	return r.start > 0
}

func (r lineRange) contains(line int) bool {
	if !r.enabled() {
		return true
	}
	return line >= r.start && line <= r.end
}

func (c filterFieldsCore) With(fields []zapcore.Field) zapcore.Core {
	return filterFieldsCore{
		Core: c.Core.With(filterFields(fields, c.drop)),
		drop: c.drop,
	}
}

func (c filterFieldsCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(ent.Level) {
		return ce
	}
	return ce.AddCore(ent, c)
}

func (c filterFieldsCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(ent, filterFields(fields, c.drop))
}

func (c debugFilterCore) With(fields []zapcore.Field) zapcore.Core {
	return debugFilterCore{
		Core:  c.Core.With(fields),
		rules: c.rules,
		layer: mergeLayer(c.layer, fields),
	}
}

func (c debugFilterCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(ent.Level) {
		return ce
	}
	return ce.AddCore(ent, c)
}

func (c debugFilterCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if !c.shouldWrite(ent, fields) {
		return nil
	}
	return c.Core.Write(ent, fields)
}

func (c debugFilterCore) shouldWrite(ent zapcore.Entry, fields []zapcore.Field) bool {
	if c.rules.empty() {
		return true
	}
	layer := mergeLayer(c.layer, fields)
	return c.rules.matches(debugEvent{
		logger:   ent.LoggerName,
		file:     ent.Caller.File,
		line:     ent.Caller.Line,
		function: ent.Caller.Function,
		layer:    layer,
	})
}

type debugEvent struct {
	logger   string
	file     string
	line     int
	function string
	layer    string
}

func (r debugAllowlist) empty() bool {
	return len(r.layers) == 0 && len(r.loggers) == 0 && len(r.files) == 0 && len(r.functions) == 0
}

func (r debugAllowlist) matches(event debugEvent) bool {
	return loggerAllowed(event.logger, r.loggers) ||
		layerAllowed(event.layer, r.layers) ||
		locationAllowed(event.file, event.line, r.files, fileValueMatches) ||
		locationAllowed(event.function, event.line, r.functions, functionValueMatches)
}

func parseDebugAllowlist(entries []string) debugAllowlist {
	rules := debugAllowlist{
		layers:  map[string]struct{}{},
		loggers: map[string]struct{}{},
	}
	for _, raw := range entries {
		kind, value, ok := splitAllowlistEntry(raw)
		if !ok {
			continue
		}
		switch kind {
		case "layer":
			if value != "" {
				rules.layers[strings.TrimSpace(value)] = struct{}{}
			}
		case "logger":
			if value != "" {
				rules.loggers[strings.TrimSpace(value)] = struct{}{}
			}
		case "file":
			base, lines, ok := splitLocationRule(value)
			if ok && base != "" {
				rules.files = append(rules.files, debugLocationRule{value: normalizeFilePath(base), lines: lines})
			}
		case "func", "function":
			base, lines, ok := splitLocationRule(value)
			if ok && base != "" {
				rules.functions = append(rules.functions, debugLocationRule{value: strings.TrimSpace(base), lines: lines})
			}
		}
	}
	if len(rules.layers) == 0 {
		rules.layers = nil
	}
	if len(rules.loggers) == 0 {
		rules.loggers = nil
	}
	return rules
}

func prefixEntries(prefix string, values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, prefix+":"+value)
	}
	return out
}

func splitAllowlistEntry(entry string) (string, string, bool) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return "", "", false
	}
	kind, value, ok := strings.Cut(entry, ":")
	if !ok {
		return "", "", false
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	value = strings.TrimSpace(value)
	if kind == "" || value == "" {
		return "", "", false
	}
	return kind, value, true
}

func splitLocationRule(value string) (string, lineRange, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", lineRange{}, false
	}
	idx := strings.LastIndex(value, ":")
	if idx == -1 {
		return value, lineRange{}, true
	}
	base := strings.TrimSpace(value[:idx])
	rng, ok := parseLineRange(value[idx+1:])
	if !ok {
		return value, lineRange{}, true
	}
	if base == "" {
		return "", lineRange{}, false
	}
	return base, rng, true
}

func parseLineRange(value string) (lineRange, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return lineRange{}, false
	}
	startText, endText, hasDash := strings.Cut(value, "-")
	if !hasDash {
		n, err := strconv.Atoi(startText)
		if err != nil || n <= 0 {
			return lineRange{}, false
		}
		return lineRange{start: n, end: n}, true
	}
	start, err := strconv.Atoi(strings.TrimSpace(startText))
	if err != nil || start <= 0 {
		return lineRange{}, false
	}
	end, err := strconv.Atoi(strings.TrimSpace(endText))
	if err != nil || end < start {
		return lineRange{}, false
	}
	return lineRange{start: start, end: end}, true
}

func makeDropSet(keys []string) map[string]struct{} {
	if len(keys) == 0 {
		return nil
	}
	keys = lo.Filter(keys, func(key string, _ int) bool { return key != "" })
	if len(keys) == 0 {
		return nil
	}
	return lo.SliceToMap(keys, func(key string) (string, struct{}) { return key, struct{}{} })
}

func filterFields(fields []zapcore.Field, drop map[string]struct{}) []zapcore.Field {
	if len(fields) == 0 || len(drop) == 0 {
		return fields
	}
	return lo.Filter(fields, func(field zapcore.Field, _ int) bool {
		_, ok := drop[field.Key]
		return !ok
	})
}

func loggerAllowed(name string, allow map[string]struct{}) bool {
	name = strings.TrimSpace(name)
	if len(allow) == 0 || name == "" {
		return false
	}
	for candidate := range allow {
		if name == candidate || strings.HasPrefix(name, candidate+".") {
			return true
		}
	}
	return false
}

func layerAllowed(name string, allow map[string]struct{}) bool {
	name = strings.TrimSpace(name)
	if len(allow) == 0 || name == "" {
		return false
	}
	for candidate := range allow {
		if name == candidate || strings.HasPrefix(name, candidate+".") {
			return true
		}
	}
	return false
}

func locationAllowed(value string, line int, rules []debugLocationRule, match func(string, string) bool) bool {
	value = strings.TrimSpace(value)
	if len(rules) == 0 || value == "" {
		return false
	}
	for _, rule := range rules {
		if !rule.lines.contains(line) {
			continue
		}
		if match(value, rule.value) {
			return true
		}
	}
	return false
}

func fileValueMatches(value string, candidate string) bool {
	value = normalizeFilePath(value)
	if value == "" || candidate == "" {
		return false
	}
	return value == candidate || strings.HasSuffix(value, "/"+candidate)
}

func functionValueMatches(value string, candidate string) bool {
	value = strings.TrimSpace(value)
	if value == "" || candidate == "" {
		return false
	}
	return value == candidate || strings.HasSuffix(value, candidate)
}

func normalizeFilePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	return path
}

func mergeLayer(current string, fields []zapcore.Field) string {
	layer := current
	for _, field := range fields {
		if field.Key != appLayerFieldKey {
			continue
		}
		if value, ok := stringFieldValue(field); ok && value != "" {
			layer = value
		}
	}
	return layer
}

func stringFieldValue(field zapcore.Field) (string, bool) {
	switch field.Type {
	case zapcore.StringType:
		return field.String, true
	case zapcore.ReflectType:
		value, ok := field.Interface.(string)
		return value, ok
	case zapcore.StringerType:
		if field.Interface == nil {
			return "", false
		}
		if s, ok := field.Interface.(interface{ String() string }); ok {
			return s.String(), true
		}
	}
	return "", false
}

func appLayerField(value string) zap.Field {
	return zap.String(appLayerFieldKey, value)
}
