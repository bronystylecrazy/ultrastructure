package di

import (
	"reflect"
	"testing"

	"go.uber.org/fx/fxtest"
)

func TestWhenParameterless(t *testing.T) {
	// When with a parameterless predicate should work without a resolver.
	RegisterResolver(nil) // clear any previous resolver
	defer RegisterResolver(nil)

	var got *basicThing
	app := fxtest.New(t,
		App(
			When(func() bool { return true },
				Provide(newBasicThing),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestWhenParameterlessFalse(t *testing.T) {
	RegisterResolver(nil)
	defer RegisterResolver(nil)

	// When returns false -> empty fx.Options(), so nothing is provided.
	// We just verify the app can start without error.
	app := fxtest.New(t,
		App(
			When(func() bool { return false },
				Provide(newBasicThing),
			),
			// No Populate since *basicThing won't be in the graph.
		).Build(),
	)
	defer app.RequireStart().RequireStop()
}

func TestWhenWithConfigResolver(t *testing.T) {
	type config struct {
		Enabled bool
	}
	cfg := &config{Enabled: true}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	var got *basicThing
	app := fxtest.New(t,
		App(
			When(func(c *config) bool { return c.Enabled },
				Provide(newBasicThing),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestWhenWithConfigResolverDisabled(t *testing.T) {
	type config struct {
		Enabled bool
	}
	cfg := &config{Enabled: false}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	// When returns false because cfg.Enabled is false -> empty options.
	app := fxtest.New(t,
		App(
			When(func(c *config) bool { return c.Enabled },
				Provide(newBasicThing),
			),
			// No Populate since *basicThing won't be in the graph.
		).Build(),
	)
	defer app.RequireStart().RequireStop()
}

func TestWhenRequiresResolverForParams(t *testing.T) {
	type config struct {
		Enabled bool
	}
	RegisterResolver(nil) // explicitly no resolver
	defer RegisterResolver(nil)

	// Build the When node directly - it should fail because resolver is nil but params are needed.
	_, err := When(func(c *config) bool { return c.Enabled },
		Provide(newBasicThing),
	).Build()
	if err == nil {
		t.Fatal("expected error when resolver is not registered for parameterized When")
	}
}

func TestWhenWithMultipleParams(t *testing.T) {
	type configA struct{ Enabled bool }
	type configB struct{ Name string }

	cfgA := &configA{Enabled: true}
	cfgB := &configB{Name: "test"}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		switch typ {
		case reflect.TypeOf(cfgA):
			return cfgA, nil
		case reflect.TypeOf(cfgB):
			return cfgB, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	var got *basicThing
	app := fxtest.New(t,
		App(
			When(func(a *configA, b *configB) bool { return a.Enabled && b.Name != "" },
				Provide(newBasicThing),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestWhenWithParamReturnsFalse(t *testing.T) {
	type config struct{ Enabled bool }
	cfg := &config{Enabled: false}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	// When returns false -> empty options.
	app := fxtest.New(t,
		App(
			When(func(c *config) bool { return c.Enabled },
				Provide(newBasicThing),
			),
			// No Populate since *basicThing won't be in the graph.
		).Build(),
	)
	defer app.RequireStart().RequireStop()
}

func TestWhenWithModule(t *testing.T) {
	type config struct{ Enabled bool }
	cfg := &config{Enabled: true}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	var got *basicThing
	app := fxtest.New(t,
		App(
			Module("inner",
				When(func(c *config) bool { return c.Enabled },
					Provide(newBasicThing),
				),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestWhenWithNestedConditionals(t *testing.T) {
	type config struct{ Enabled bool }
	cfg := &config{Enabled: true}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	var got *basicThing
	app := fxtest.New(t,
		App(
			When(func(c *config) bool { return c.Enabled },
				When(func() bool { return true },
					Provide(newBasicThing),
				),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestWhenWithModuleScopesResolver(t *testing.T) {
	type config struct{ Enabled bool }
	cfg := &config{Enabled: true}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	// When inside a module should still receive the resolver.
	var got *basicThing
	app := fxtest.New(t,
		App(
			Module("child",
				When(func(c *config) bool { return c.Enabled },
					Provide(newBasicThing),
				),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestWhenFunctionNil(t *testing.T) {
	RegisterResolver(nil)
	defer RegisterResolver(nil)

	// Build the When node with nil function - it should return an error.
	_, err := When(nil, Provide(newBasicThing)).Build()
	if err == nil {
		t.Fatal("expected error for nil function")
	}
}

func TestWhenInvalidFunctionNotBool(t *testing.T) {
	RegisterResolver(nil)
	defer RegisterResolver(nil)

	// Build the When node with a function that returns non-bool - it should return an error.
	_, err := When(func() string { return "not bool" }, Provide(newBasicThing)).Build()
	if err == nil {
		t.Fatal("expected error for non-bool returning function")
	}
}

func TestWhenCachesResult(t *testing.T) {
	type config struct{ Counter int; Enabled bool }
	cfg := &config{Counter: 0, Enabled: true}

	resolver := func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			cfg.Counter++ // increment to verify caching
			return cfg, nil
		}
		return nil, nil
	}
	RegisterResolver(resolver)
	defer RegisterResolver(nil)

	var got *basicThing
	app := fxtest.New(t,
		App(
			When(func(c *config) bool { return c.Enabled },
				Provide(newBasicThing),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	// Counter should be 1, not 2, because eval caches its result.
	if cfg.Counter != 1 {
		t.Fatalf("expected counter=1 (cached), got: %d", cfg.Counter)
	}
}

func TestSwitchWithWhenCase(t *testing.T) {
	type config struct{ Mode string }
	cfg := &config{Mode: "b"}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	var got string
	app := fxtest.New(t,
		App(
			Switch(
				WhenCase(func(c *config) bool { return c.Mode == "a" },
					Supply(&basicThing{value: "a"}),
				),
				WhenCase(func(c *config) bool { return c.Mode == "b" },
					Supply(&basicThing{value: "b"}),
				),
				DefaultCase(
					Supply(&basicThing{value: "default"}),
				),
			),
			Invoke(func(b *basicThing) {
				got = b.value
			}),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "b" {
		t.Fatalf("expected 'b', got: %q", got)
	}
}

func TestSwitchWithWhenCaseNoMatchFallsThrough(t *testing.T) {
	type config struct{ Mode string }
	cfg := &config{Mode: "c"}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	var got string
	app := fxtest.New(t,
		App(
			Switch(
				WhenCase(func(c *config) bool { return c.Mode == "a" },
					Supply(&basicThing{value: "a"}),
				),
				WhenCase(func(c *config) bool { return c.Mode == "b" },
					Supply(&basicThing{value: "b"}),
				),
				DefaultCase(
					Supply(&basicThing{value: "default"}),
				),
			),
			Invoke(func(b *basicThing) {
				got = b.value
			}),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "default" {
		t.Fatalf("expected 'default', got: %q", got)
	}
}

func TestSwitchWithMixedCaseAndWhenCase(t *testing.T) {
	type config struct{ Flag bool }
	cfg := &config{Flag: true}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	var got string
	app := fxtest.New(t,
		App(
			Switch(
				Case(false, // static false - skipped
					Supply(&basicThing{value: "a"}),
				),
				WhenCase(func(c *config) bool { return c.Flag },
					Supply(&basicThing{value: "flag_true"}),
				),
				DefaultCase(
					Supply(&basicThing{value: "default"}),
				),
			),
			Invoke(func(b *basicThing) {
				got = b.value
			}),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "flag_true" {
		t.Fatalf("expected 'flag_true', got: %q", got)
	}
}

func TestWhenCaseWithModule(t *testing.T) {
	type config struct{ Enabled bool }
	cfg := &config{Enabled: true}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	var got *basicThing
	app := fxtest.New(t,
		App(
			Switch(
				WhenCase(func(c *config) bool { return c.Enabled },
					Module("inner",
						Supply(&basicThing{value: "from_module"}),
					),
				),
				DefaultCase(
					Supply(&basicThing{value: "default"}),
				),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "from_module" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestWhenCaseResolverNotRegistered(t *testing.T) {
	type config struct{ Enabled bool }
	RegisterResolver(nil) // explicitly no resolver
	defer RegisterResolver(nil)

	// WhenCase with a config parameter but no resolver registered.
	// The error surfaces at Build() time when evalWhen is called.
	// WhenCase returns caseNode which has no Build(); wrap in Switch to build.
	node := Switch(WhenCase(func(c *config) bool { return c.Enabled },
		Provide(newBasicThing),
	))
	_, err := node.Build()
	if err == nil {
		t.Fatal("expected error when resolver is not registered for parameterized WhenCase")
	}
}

func TestWhenResolverNotRegistered(t *testing.T) {
	type config struct{ Enabled bool }
	RegisterResolver(nil) // explicitly no resolver
	defer RegisterResolver(nil)

	// When with a config parameter but no resolver registered.
	node := When(func(c *config) bool { return c.Enabled },
		Provide(newBasicThing),
	)
	_, err := node.Build()
	if err == nil {
		t.Fatal("expected error when resolver is not registered for parameterized When")
	}
}

func TestWhenWithViperLikeResolver(t *testing.T) {
	// Simulate a viper-like resolver that resolves from a map.
	settings := map[string]any{
		"feature.enabled": true,
		"feature.name":   "test",
	}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		switch typ.Kind() {
		case reflect.Bool:
			if typ.Name() == "bool" {
				return settings["feature.enabled"], nil
			}
		case reflect.String:
			if typ.Name() == "string" {
				return settings["feature.name"], nil
			}
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	var got *basicThing
	app := fxtest.New(t,
		App(
			When(func() bool { return settings["feature.enabled"].(bool) },
				Provide(newBasicThing),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestWhenWithEightParams(t *testing.T) {
	type (
		T0 struct{ V bool }
		T1 struct{ V bool }
		T2 struct{ V bool }
		T3 struct{ V bool }
		T4 struct{ V bool }
		T5 struct{ V bool }
		T6 struct{ V bool }
		T7 struct{ V bool }
	)

	v0, v1, v2, v3, v4, v5, v6, v7 := &T0{V: true}, &T1{V: true}, &T2{V: true}, &T3{V: true}, &T4{V: true}, &T5{V: true}, &T6{V: true}, &T7{V: true}

	RegisterResolver(func(typ reflect.Type) (any, error) {
		switch typ {
		case reflect.TypeOf(v0): return v0, nil
		case reflect.TypeOf(v1): return v1, nil
		case reflect.TypeOf(v2): return v2, nil
		case reflect.TypeOf(v3): return v3, nil
		case reflect.TypeOf(v4): return v4, nil
		case reflect.TypeOf(v5): return v5, nil
		case reflect.TypeOf(v6): return v6, nil
		case reflect.TypeOf(v7): return v7, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	var got *basicThing
	app := fxtest.New(t,
		App(
			When(func(a0 *T0, a1 *T1, a2 *T2, a3 *T3, a4 *T4, a5 *T5, a6 *T6, a7 *T7) bool {
				return a0.V && a1.V && a2.V && a3.V && a4.V && a5.V && a6.V && a7.V
			},
				Provide(newBasicThing),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestWhenTooManyParams(t *testing.T) {
	type T0 struct{ V bool }
	type T1 struct{ V bool }
	type T2 struct{ V bool }
	type T3 struct{ V bool }
	type T4 struct{ V bool }
	type T5 struct{ V bool }
	type T6 struct{ V bool }
	type T7 struct{ V bool }
	type T8 struct{ V bool } // 9th param - over limit

	RegisterResolver(nil)
	defer RegisterResolver(nil)

	node := When(func(a0 *T0, a1 *T1, a2 *T2, a3 *T3, a4 *T4, a5 *T5, a6 *T6, a7 *T7, a8 *T8) bool {
		return true
	}, Provide(newBasicThing))

	_, err := node.Build()
	if err == nil {
		t.Fatal("expected error for more than 8 parameters")
	}
}

func TestWhenNotAFunction(t *testing.T) {
	RegisterResolver(nil)
	defer RegisterResolver(nil)

	node := When("not a function", Provide(newBasicThing))
	_, err := node.Build()
	if err == nil {
		t.Fatal("expected error for non-function argument")
	}
}

func TestWhenDoesNotReturnBool(t *testing.T) {
	RegisterResolver(nil)
	defer RegisterResolver(nil)

	node := When(func() int { return 42 }, Provide(newBasicThing))
	_, err := node.Build()
	if err == nil {
		t.Fatal("expected error for non-bool returning function")
	}
}

func TestWhenNilResolverWithNoParams(t *testing.T) {
	// Should not error even with nil resolver, since no params need resolution.
	RegisterResolver(nil)
	defer RegisterResolver(nil)

	var got *basicThing
	app := fxtest.New(t,
		App(
			When(func() bool { return true },
				Provide(newBasicThing),
			),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestSwitchWhenCaseDefaultCaseFallsThrough(t *testing.T) {
	type config struct{ Mode string }
	cfg := &config{Mode: "z"} // doesn't match any case

	RegisterResolver(func(typ reflect.Type) (any, error) {
		if typ == reflect.TypeOf(cfg) {
			return cfg, nil
		}
		return nil, nil
	})
	defer RegisterResolver(nil)

	// No DefaultCase and no case matches -> Switch returns empty fx.Options().
	// No Invoke or Populate that depends on *basicThing.
	app := fxtest.New(t,
		App(
			Switch(
				WhenCase(func(c *config) bool { return c.Mode == "a" },
					Supply(&basicThing{value: "a"}),
				),
				WhenCase(func(c *config) bool { return c.Mode == "b" },
					Supply(&basicThing{value: "b"}),
				),
				// No DefaultCase - no match means empty graph.
			),
		).Build(),
	)
	defer app.RequireStart().RequireStop()
}
