package web

import (
	"reflect"
	"strings"

	"github.com/Flussen/swagger-fiber-v3"
	"github.com/gofiber/fiber/v3"
)

type SwaggerOption func(*swaggerOptions)

type SwaggerMiddleware struct {
	config Config
	path   string
}

func NewSwaggerMiddleware(config Config) (*SwaggerMiddleware, error) {
	return NewSwaggerMiddlewareWithOptions(WithSwaggerConfig(config))
}

type swaggerOptions struct {
	config          Config
	path            string
	securitySchemes map[string]interface{}
	defaultSecurity []SecurityRequirement
	tagDescriptions map[string]string
	termsOfService  string
	contact         *AutoSwaggerContact
	license         *AutoSwaggerLicense
	hook            HookFunc
	extraModels     []reflect.Type
}

func WithSwaggerConfig(config Config) SwaggerOption {
	return func(o *swaggerOptions) {
		o.config = config
	}
}

func WithSwaggerPath(path string) SwaggerOption {
	return func(o *swaggerOptions) {
		o.path = path
	}
}

// WithBearerSecurityScheme adds a global HTTP bearer security scheme to auto-generated OpenAPI docs.
func WithBearerSecurityScheme(name string) SwaggerOption {
	return func(o *swaggerOptions) {
		if name == "" {
			name = "BearerAuth"
		}
		ensureSecuritySchemesMap(o)
		o.securitySchemes[name] = map[string]interface{}{
			"type":         "http",
			"scheme":       "bearer",
			"bearerFormat": "JWT",
		}
	}
}

// WithAPIKeySecurityScheme adds a global API key security scheme to auto-generated OpenAPI docs.
// Parameter "in" should be one of: header, query, cookie.
func WithAPIKeySecurityScheme(name, keyName, in string) SwaggerOption {
	return func(o *swaggerOptions) {
		if name == "" {
			name = "ApiKeyAuth"
		}
		if keyName == "" {
			keyName = "X-API-Key"
		}
		switch in {
		case "query", "cookie":
			// valid
		default:
			in = "header"
		}

		ensureSecuritySchemesMap(o)
		o.securitySchemes[name] = map[string]interface{}{
			"type": "apiKey",
			"name": keyName,
			"in":   in,
		}
	}
}

// WithDefaultSecurity adds a global default OpenAPI security requirement.
// Usage: WithDefaultSecurity("BearerAuth") or WithDefaultSecurity("OAuth2", "read:users")
func WithDefaultSecurity(scheme string, scopes ...string) SwaggerOption {
	return func(o *swaggerOptions) {
		scheme = strings.TrimSpace(scheme)
		if scheme == "" {
			return
		}
		o.defaultSecurity = append(o.defaultSecurity, SecurityRequirement{
			Scheme: scheme,
			Scopes: append([]string(nil), scopes...),
		})
	}
}

// WithTagDescription adds a top-level OpenAPI tag with description (or enriches an existing tag).
func WithTagDescription(name, description string) SwaggerOption {
	return func(o *swaggerOptions) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if o.tagDescriptions == nil {
			o.tagDescriptions = make(map[string]string)
		}
		o.tagDescriptions[name] = description
	}
}

// WithAPITermsOfService sets OpenAPI info.termsOfService.
func WithAPITermsOfService(url string) SwaggerOption {
	return func(o *swaggerOptions) {
		o.termsOfService = strings.TrimSpace(url)
	}
}

// WithAPIContact sets OpenAPI info.contact.
func WithAPIContact(name, url, email string) SwaggerOption {
	return func(o *swaggerOptions) {
		o.contact = &AutoSwaggerContact{
			Name:  strings.TrimSpace(name),
			URL:   strings.TrimSpace(url),
			Email: strings.TrimSpace(email),
		}
	}
}

// WithAPILicense sets OpenAPI info.license.
func WithAPILicense(name, url string) SwaggerOption {
	return func(o *swaggerOptions) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		o.license = &AutoSwaggerLicense{
			Name: name,
			URL:  strings.TrimSpace(url),
		}
	}
}

// WithSwaggerCustomize registers a per-UseAutoSwagger metadata hook.
// This hook can mutate operation metadata before OpenAPI operation generation.
func WithSwaggerCustomize(hook HookFunc) SwaggerOption {
	return func(o *swaggerOptions) {
		if hook == nil {
			return
		}
		if o.hook == nil {
			o.hook = hook
			return
		}
		prev := o.hook
		o.hook = func(ctx *SwaggerContext) {
			prev(ctx)
			hook(ctx)
		}
	}
}

// RegisterHook sets the autoswagger hook inside UseAutoSwagger(...).
// Usage: UseAutoSwagger(RegisterHook(func(ctx *SwaggerContext) { ... }))
// When called with no arguments it is a no-op option.
func RegisterHook(hook ...HookFunc) SwaggerOption {
	if len(hook) == 0 || hook[0] == nil {
		return func(o *swaggerOptions) {}
	}
	return WithSwaggerCustomize(hook[0])
}

// WithSwaggerExtraModels registers additional model types to always include
// in OpenAPI components.schemas generation.
func WithSwaggerExtraModels(models ...any) SwaggerOption {
	return func(o *swaggerOptions) {
		if len(models) == 0 {
			return
		}

		if o.extraModels == nil {
			o.extraModels = make([]reflect.Type, 0, len(models))
		}

		for _, model := range models {
			t := normalizeSwaggerModelInput(model)
			if t == nil {
				continue
			}
			o.extraModels = append(o.extraModels, t)
		}
	}
}

func ensureSecuritySchemesMap(o *swaggerOptions) {
	if o.securitySchemes == nil {
		o.securitySchemes = make(map[string]interface{})
	}
}

func NewSwaggerMiddlewareWithOptions(opts ...SwaggerOption) (*SwaggerMiddleware, error) {
	cfg := swaggerOptions{path: "/docs/*"}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &SwaggerMiddleware{config: cfg.config, path: cfg.path}, nil
}

func (h *SwaggerMiddleware) Handle(r fiber.Router) {
	r.Get(h.path, swagger.HandlerDefault)
}
