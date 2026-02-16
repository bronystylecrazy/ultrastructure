package web

import (
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
