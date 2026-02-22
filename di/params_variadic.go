package di

import (
	"fmt"
	"reflect"
)

func resolveVariadicBindParams(function any, cfg *bindConfig) error {
	if cfg == nil || !cfg.variadicShorthand {
		return nil
	}
	fnType := reflect.TypeOf(function)
	if fnType == nil || fnType.Kind() != reflect.Func || !fnType.IsVariadic() {
		return fmt.Errorf(errVariadicRequiresVariadicTarget)
	}
	expanded, err := expandVariadicShorthandTags(fnType.NumIn(), cfg.paramTags)
	if err != nil {
		return err
	}
	cfg.paramTags = expanded
	cfg.paramSlots = fnType.NumIn()
	cfg.variadicShorthand = false
	return nil
}

func resolveVariadicParamTags(function any, cfg *paramConfig, expected int) error {
	if cfg == nil || !cfg.variadicShorthand {
		return nil
	}
	fnType := reflect.TypeOf(function)
	if fnType == nil || fnType.Kind() != reflect.Func || !fnType.IsVariadic() {
		return fmt.Errorf(errVariadicRequiresVariadicTarget)
	}
	expanded, err := expandVariadicShorthandTags(expected, cfg.tags)
	if err != nil {
		return err
	}
	cfg.tags = expanded
	cfg.paramSlots = expected
	cfg.variadicShorthand = false
	return nil
}

func expandVariadicShorthandTags(expected int, tags []string) ([]string, error) {
	if expected <= 0 {
		return nil, fmt.Errorf(errVariadicRequiresVariadicTarget)
	}
	if len(tags) != 1 {
		return nil, fmt.Errorf(errVariadicParamTagSingle, len(tags))
	}
	out := make([]string, expected)
	out[expected-1] = tags[0]
	return out, nil
}
