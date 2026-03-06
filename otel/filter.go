package otel

import (
	"github.com/samber/lo"
	"go.uber.org/zap/zapcore"
)

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
	keys = lo.Filter(keys, func(key string, _ int) bool { return key != "" })
	if len(keys) == 0 {
		return nil
	}
	drop := lo.SliceToMap(keys, func(key string) (string, struct{}) { return key, struct{}{} })
	return drop
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
