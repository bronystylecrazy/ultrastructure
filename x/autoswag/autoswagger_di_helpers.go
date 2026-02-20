package autoswag

import "reflect"

func combineExtraModelTypes(registry *SwaggerModelRegistry, optionModels []reflect.Type) []reflect.Type {
	combined := make([]reflect.Type, 0, len(optionModels))
	seen := make(map[reflect.Type]struct{}, len(optionModels))

	if registry != nil {
		for _, t := range registry.Types() {
			normalized := normalizeModelType(t)
			if normalized == nil {
				continue
			}
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			combined = append(combined, normalized)
		}
	}

	for _, t := range optionModels {
		normalized := normalizeModelType(t)
		if normalized == nil {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		combined = append(combined, normalized)
	}

	return combined
}

type swaggerCustomizeHooks struct {
	pre  HookFunc
	run  HookFunc
	post HookFunc
}

func composeSwaggerCustomizeHooks(
	base HookFunc,
	customizers []Customizer,
	preCustomizers []PreRun,
	postCustomizers []PostRun,
) swaggerCustomizeHooks {
	var out swaggerCustomizeHooks

	if len(preCustomizers) > 0 {
		out.pre = func(ctx *Context) {
			for _, customizer := range preCustomizers {
				if customizer == nil {
					continue
				}
				customizer.PreCustomizeSwagger(ctx)
			}
		}
	}

	if len(postCustomizers) > 0 {
		out.post = func(ctx *Context) {
			for _, customizer := range postCustomizers {
				if customizer == nil {
					continue
				}
				customizer.PostCustomizeSwagger(ctx)
			}
		}
	}

	if base == nil && len(customizers) == 0 {
		return out
	}

	out.run = func(ctx *Context) {
		if base != nil {
			base(ctx)
		}
		for _, customizer := range customizers {
			if customizer == nil {
				continue
			}
			customizer.CustomizeSwagger(ctx)
		}
	}

	return out
}
