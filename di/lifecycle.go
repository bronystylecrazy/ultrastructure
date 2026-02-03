package di

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/fx"
)

// OnStart registers a lifecycle OnStart hook.
func OnStart(fn any) Node {
	return lifecycleNode{kind: lifecycleStart, fn: fn}
}

// OnStop registers a lifecycle OnStop hook.
func OnStop(fn any) Node {
	return lifecycleNode{kind: lifecycleStop, fn: fn}
}

type lifecycleKind int

const (
	lifecycleStart lifecycleKind = iota
	lifecycleStop
)

type lifecycleNode struct {
	kind lifecycleKind
	fn   any
}

func (n lifecycleNode) Build() (fx.Option, error) {
	fnType := reflect.TypeOf(n.fn)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf(errLifecycleHookFunction)
	}
	if fnType.NumIn() > 1 {
		return nil, fmt.Errorf(errLifecycleHookParamCount)
	}
	if fnType.NumOut() > 1 {
		return nil, fmt.Errorf(errLifecycleHookReturnCount)
	}
	if fnType.NumOut() == 1 && fnType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		return nil, fmt.Errorf(errLifecycleHookReturnType)
	}
	if fnType.NumIn() == 1 {
		param := fnType.In(0)
		ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
		lifecycleType := reflect.TypeOf((*fx.Lifecycle)(nil)).Elem()
		if param != ctxType && param != lifecycleType {
			return nil, fmt.Errorf(errLifecycleHookParamType)
		}
	}

	makeHook := func(lc fx.Lifecycle) func(context.Context) error {
		return func(ctx context.Context) error {
			var args []reflect.Value
			// Support hooks taking either context.Context or fx.Lifecycle.
			if fnType.NumIn() == 1 {
				param := fnType.In(0)
				if param == reflect.TypeOf((*fx.Lifecycle)(nil)).Elem() {
					args = append(args, reflect.ValueOf(lc))
				} else {
					args = append(args, reflect.ValueOf(ctx))
				}
			}
			out := reflect.ValueOf(n.fn).Call(args)
			if len(out) == 1 && !out[0].IsNil() {
				return out[0].Interface().(error)
			}
			return nil
		}
	}

	return fx.Invoke(func(lc fx.Lifecycle) {
		hookFn := makeHook(lc)
		hook := fx.Hook{}
		if n.kind == lifecycleStart {
			hook.OnStart = hookFn
		} else {
			hook.OnStop = hookFn
		}
		lc.Append(hook)
	}), nil
}
