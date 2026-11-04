package otel

import (
	"bytes"
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"
)

// copyingCore gives the core it wraps entries that own their string and byte
// data.
//
// zap's encoders are done with an entry when Write returns, so callers may log
// strings that alias reusable buffers, as Fiber and fasthttp do with request
// data. The OpenTelemetry bridge keeps those strings as they are and exports
// them later from a batch, by which time a later request may have overwritten
// them. Copying before the bridge sees them keeps the exported record intact.
type copyingCore struct {
	zapcore.Core
}

func newCopyingCore(core zapcore.Core) zapcore.Core {
	return copyingCore{Core: core}
}

func (c copyingCore) With(fields []zapcore.Field) zapcore.Core {
	return copyingCore{Core: c.Core.With(copyFields(fields))}
}

func (c copyingCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(ent.Level) {
		return ce
	}
	return ce.AddCore(ent, c)
}

func (c copyingCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	ent.Message = strings.Clone(ent.Message)
	return c.Core.Write(ent, copyFields(fields))
}

// copyFields copies fields into a new slice, since a tee hands the caller's
// slice to its other cores as well.
func copyFields(fields []zapcore.Field) []zapcore.Field {
	if len(fields) == 0 {
		return fields
	}
	out := make([]zapcore.Field, len(fields))
	for i, f := range fields {
		out[i] = copyField(f)
	}
	return out
}

func copyField(f zapcore.Field) zapcore.Field {
	f.Key = strings.Clone(f.Key)
	switch f.Type {
	case zapcore.StringType:
		f.String = strings.Clone(f.String)
	case zapcore.ByteStringType, zapcore.BinaryType, zapcore.ReflectType:
		f.Interface = copyValue(f.Interface)
	case zapcore.StringerType:
		if s, ok := f.Interface.(fmt.Stringer); ok {
			if str, ok := stringOf(s); ok {
				f.Type, f.String, f.Interface = zapcore.StringType, strings.Clone(str), nil
			}
		}
	case zapcore.ArrayMarshalerType:
		if m, ok := f.Interface.(zapcore.ArrayMarshaler); ok {
			f.Interface = copyingArray{m}
		}
	case zapcore.ObjectMarshalerType, zapcore.InlineMarshalerType:
		if m, ok := f.Interface.(zapcore.ObjectMarshaler); ok {
			f.Interface = copyingObject{m}
		}
	}
	return f
}

// copyValue copies strings and byte slices; other values are kept as they are.
func copyValue(v any) any {
	switch v := v.(type) {
	case string:
		return strings.Clone(v)
	case []byte:
		return bytes.Clone(v)
	}
	return v
}

// stringOf calls String, reporting false if it panics so that the field is
// left for zap, which renders a nil receiver as "<nil>".
func stringOf(s fmt.Stringer) (str string, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	return s.String(), true
}

// copyingArray and copyingObject hand their marshalers an encoder that copies
// what is added to it, which covers fields such as zap.Strings.
type copyingArray struct{ zapcore.ArrayMarshaler }

func (a copyingArray) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	return a.ArrayMarshaler.MarshalLogArray(copyingArrayEncoder{enc})
}

type copyingObject struct{ zapcore.ObjectMarshaler }

func (o copyingObject) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return o.ObjectMarshaler.MarshalLogObject(copyingObjectEncoder{enc})
}

type copyingObjectEncoder struct{ zapcore.ObjectEncoder }

func (e copyingObjectEncoder) AddString(k, v string) {
	e.ObjectEncoder.AddString(k, strings.Clone(v))
}

func (e copyingObjectEncoder) AddByteString(k string, v []byte) {
	e.ObjectEncoder.AddByteString(k, bytes.Clone(v))
}

func (e copyingObjectEncoder) AddBinary(k string, v []byte) {
	e.ObjectEncoder.AddBinary(k, bytes.Clone(v))
}

func (e copyingObjectEncoder) AddReflected(k string, v any) error {
	return e.ObjectEncoder.AddReflected(k, copyValue(v))
}

func (e copyingObjectEncoder) AddArray(k string, v zapcore.ArrayMarshaler) error {
	return e.ObjectEncoder.AddArray(k, copyingArray{v})
}

func (e copyingObjectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	return e.ObjectEncoder.AddObject(k, copyingObject{v})
}

type copyingArrayEncoder struct{ zapcore.ArrayEncoder }

func (e copyingArrayEncoder) AppendString(v string) {
	e.ArrayEncoder.AppendString(strings.Clone(v))
}

func (e copyingArrayEncoder) AppendByteString(v []byte) {
	e.ArrayEncoder.AppendByteString(bytes.Clone(v))
}

func (e copyingArrayEncoder) AppendReflected(v any) error {
	return e.ArrayEncoder.AppendReflected(copyValue(v))
}

func (e copyingArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	return e.ArrayEncoder.AppendArray(copyingArray{v})
}

func (e copyingArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	return e.ArrayEncoder.AppendObject(copyingObject{v})
}
