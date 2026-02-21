package autoswag

import (
	"reflect"
	"strings"
)

type Option func(*options)

const defaultEmitFilePath = "./docs/openapi.json"

type options struct {
	config              Config
	path                string
	emitFiles           []string
	versionedDocs       []VersionedDocsOption
	securitySchemes     map[string]interface{}
	defaultSecurity     []SecurityRequirement
	tagDescriptions     map[string]string
	packageTagTransform func(string) string
	includeDiagnostics  bool
	diagnosticsSeverity string
	failOnDiagnostics   bool
	termsOfService      string
	contact             *OpenAPIContact
	license             *OpenAPILicense
	hook                HookFunc
	extraModels         []reflect.Type
}

type ResolvedOptions struct {
	Config              Config
	Path                string
	EmitFiles           []string
	VersionedDocs       []VersionedDocsOption
	SecuritySchemes     map[string]interface{}
	DefaultSecurity     []SecurityRequirement
	TagDescriptions     map[string]string
	PackageTagTransform func(string) string
	IncludeDiagnostics  bool
	DiagnosticsSeverity string
	FailOnDiagnostics   bool
	TermsOfService      string
	Contact             *OpenAPIContact
	License             *OpenAPILicense
	Hook                HookFunc
	ExtraModels         []reflect.Type
}

type VersionedDocsOption struct {
	Path   string
	Prefix string
	Name   string
}

func ResolveOptions(defaultPath string, opts ...Option) ResolvedOptions {
	cfg := options{path: defaultPath}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return ResolvedOptions{
		Config:              cfg.config,
		Path:                cfg.path,
		EmitFiles:           append([]string(nil), cfg.emitFiles...),
		VersionedDocs:       append([]VersionedDocsOption(nil), cfg.versionedDocs...),
		SecuritySchemes:     cfg.securitySchemes,
		DefaultSecurity:     append([]SecurityRequirement(nil), cfg.defaultSecurity...),
		TagDescriptions:     cfg.tagDescriptions,
		PackageTagTransform: cfg.packageTagTransform,
		IncludeDiagnostics:  cfg.includeDiagnostics,
		DiagnosticsSeverity: cfg.diagnosticsSeverity,
		FailOnDiagnostics:   cfg.failOnDiagnostics,
		TermsOfService:      cfg.termsOfService,
		Contact:             cfg.contact,
		License:             cfg.license,
		Hook:                cfg.hook,
		ExtraModels:         append([]reflect.Type(nil), cfg.extraModels...),
	}
}

func WithConfig(config Config) Option {
	return func(o *options) { o.config = config }
}

func WithPath(path string) Option {
	return func(o *options) { o.path = path }
}

// WithEmitFiles writes the generated OpenAPI spec to file paths in development mode only.
// File format is selected by extension: .json, .yaml, .yml.
func WithEmitFiles(paths ...string) Option {
	return func(o *options) {
		if len(paths) == 0 {
			o.emitFiles = append(o.emitFiles, defaultEmitFilePath)
			return
		}
		before := len(o.emitFiles)
		for _, path := range paths {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			o.emitFiles = append(o.emitFiles, path)
		}
		if len(o.emitFiles) == before {
			o.emitFiles = append(o.emitFiles, defaultEmitFilePath)
		}
	}
}

// WithEmitFile is a compatibility wrapper around WithEmitFiles(path).
func WithEmitFile(path string) Option {
	return WithEmitFiles(path)
}

// WithVersionedDocs mounts an additional Swagger UI/spec for a route prefix.
// Example: WithVersionedDocs("/docs/v1", "/api/v1", "API v1")
func WithVersionedDocs(path, prefix, name string) Option {
	return func(o *options) {
		path = normalizeDocsPath(path)
		prefix = normalizeRoutePrefix(prefix)
		if path == "" || prefix == "" {
			return
		}
		o.versionedDocs = append(o.versionedDocs, VersionedDocsOption{
			Path:   path,
			Prefix: prefix,
			Name:   strings.TrimSpace(name),
		})
	}
}

func WithBearerSecurityScheme(name string) Option {
	return func(o *options) {
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

func WithAPIKeySecurityScheme(name, keyName, in string) Option {
	return func(o *options) {
		if name == "" {
			name = "ApiKeyAuth"
		}
		if keyName == "" {
			keyName = "X-API-Key"
		}
		switch in {
		case "query", "cookie":
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

func WithDefaultSecurity(scheme string, scopes ...string) Option {
	return func(o *options) {
		if scheme == "" {
			return
		}
		o.defaultSecurity = append(o.defaultSecurity, SecurityRequirement{
			Scheme: scheme,
			Scopes: append([]string(nil), scopes...),
		})
	}
}

func WithTagDescription(name, description string) Option {
	return func(o *options) {
		if name == "" {
			return
		}
		if o.tagDescriptions == nil {
			o.tagDescriptions = map[string]string{}
		}
		o.tagDescriptions[name] = description
	}
}

func WithPackageTagTransform(transform func(string) string) Option {
	return func(o *options) {
		o.packageTagTransform = transform
	}
}

// WithConflictDiagnostics includes metadata conflict diagnostics in operations as x-autoswag-warnings.
func WithConflictDiagnostics(enabled bool) Option {
	return func(o *options) {
		o.includeDiagnostics = enabled
		if enabled && strings.TrimSpace(o.diagnosticsSeverity) == "" {
			o.diagnosticsSeverity = "warning"
		}
	}
}

// WithConflictDiagnosticsSeverity sets conflict severity: "warning" (default) or "error".
func WithConflictDiagnosticsSeverity(severity string) Option {
	return func(o *options) {
		o.diagnosticsSeverity = normalizeDiagnosticSeverity(severity)
	}
}

// WithFailOnConflictDiagnostics panics during spec build when an "error" diagnostic is present.
func WithFailOnConflictDiagnostics(enabled bool) Option {
	return func(o *options) {
		o.failOnDiagnostics = enabled
	}
}

func WithTermsOfService(url string) Option {
	return func(o *options) { o.termsOfService = url }
}

func WithContact(name, url, email string) Option {
	return func(o *options) {
		o.contact = &OpenAPIContact{Name: name, URL: url, Email: email}
	}
}

func WithLicense(name, url string) Option {
	return func(o *options) {
		if name == "" {
			return
		}
		o.license = &OpenAPILicense{Name: name, URL: url}
	}
}

func WithCustomize(hook HookFunc) Option {
	return func(o *options) {
		if hook == nil {
			return
		}
		if o.hook == nil {
			o.hook = hook
			return
		}
		prev := o.hook
		o.hook = func(ctx *Context) {
			prev(ctx)
			hook(ctx)
		}
	}
}

func RegisterHook(hook ...HookFunc) Option {
	if len(hook) == 0 || hook[0] == nil {
		return nil
	}
	return WithCustomize(hook[0])
}

func WithExtraModels(models ...any) Option {
	return func(o *options) {
		if len(models) == 0 {
			return
		}

		if o.extraModels == nil {
			o.extraModels = make([]reflect.Type, 0, len(models))
		}

		for _, model := range models {
			t := normalizeModelInput(model)
			if t == nil {
				continue
			}
			o.extraModels = append(o.extraModels, t)
		}
	}
}

func ensureSecuritySchemesMap(o *options) {
	if o.securitySchemes == nil {
		o.securitySchemes = make(map[string]interface{})
	}
}

func normalizeModelInput(model any) reflect.Type {
	if model == nil {
		return nil
	}
	if t, ok := model.(reflect.Type); ok {
		return normalizeModelType(t)
	}
	return normalizeModelType(reflect.TypeOf(model))
}

func normalizeModelType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func normalizeDocsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func normalizeRoutePrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if len(prefix) > 1 {
		prefix = strings.TrimSuffix(prefix, "/")
	}
	return prefix
}
