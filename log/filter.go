package log

import "go.uber.org/zap/zapcore"

// FilterFieldsCore drops fields with matching keys before writing to the core.
func FilterFieldsCore(core zapcore.Core, dropKeys ...string) zapcore.Core {
	return filterFieldsCore{
		Core: core,
		drop: makeDropSet(dropKeys),
	}
}

type filterFieldsCore struct {
	zapcore.Core
	drop map[string]struct{}
}

func (c filterFieldsCore) With(fields []zapcore.Field) zapcore.Core {
	return filterFieldsCore{
		Core: c.Core.With(filterFields(fields, c.drop)),
		drop: c.drop,
	}
}

func (c filterFieldsCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(ent, filterFields(fields, c.drop))
}

func makeDropSet(keys []string) map[string]struct{} {
	if len(keys) == 0 {
		return nil
	}
	drop := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		drop[key] = struct{}{}
	}
	return drop
}

func filterFields(fields []zapcore.Field, drop map[string]struct{}) []zapcore.Field {
	if len(fields) == 0 || len(drop) == 0 {
		return fields
	}
	out := fields[:0]
	for _, field := range fields {
		if _, ok := drop[field.Key]; ok {
			continue
		}
		out = append(out, field)
	}
	return out
}
