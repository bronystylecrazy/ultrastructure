package autoswag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/meta"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	"gopkg.in/yaml.v3"
	"go.uber.org/zap"
)

// Middleware provides runtime OpenAPI spec generation.
type Middleware struct {
	config              web.Config
	path                string
	emitFiles           []string
	versionedDocs       []VersionedDocsOption
	registry            *web.MetadataRegistry
	modelRegistry       *SwaggerModelRegistry
	extraModels         []reflect.Type
	hook                HookFunc
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
	spec                *OpenAPISpec
	versionedSpecs      []mountedSpec
	logger              *zap.Logger
}

type mountedSpec struct {
	path string
	spec *OpenAPISpec
}

func Use(opts ...Option) di.Node {
	return di.Options(
		di.AutoGroup[Customizer](CustomizersGroupName),
		di.AutoGroup[PreRun](PreCustomizersGroupName),
		di.AutoGroup[PostRun](PostCustomizersGroupName),
		di.Provide(NewSwaggerModelRegistry),
		di.Provide(func(config web.Config, logger *zap.Logger, registries *web.RegistryContainer, modelRegistry *SwaggerModelRegistry) (*Middleware, error) {
			cfg := ResolveOptions("/docs", opts...)

			var metadataRegistry *web.MetadataRegistry
			if registries != nil {
				metadataRegistry = registries.Metadata
			}

			return &Middleware{
				config:              config,
				path:                cfg.Path,
				emitFiles:           append([]string(nil), cfg.EmitFiles...),
				versionedDocs:       append([]VersionedDocsOption(nil), cfg.VersionedDocs...),
				registry:            metadataRegistry,
				modelRegistry:       modelRegistry,
				extraModels:         append([]reflect.Type(nil), cfg.ExtraModels...),
				hook:                cfg.Hook,
				securitySchemes:     cfg.SecuritySchemes,
				defaultSecurity:     append([]SecurityRequirement(nil), cfg.DefaultSecurity...),
				tagDescriptions:     cfg.TagDescriptions,
				packageTagTransform: cfg.PackageTagTransform,
				includeDiagnostics:  cfg.IncludeDiagnostics,
				diagnosticsSeverity: cfg.DiagnosticsSeverity,
				failOnDiagnostics:   cfg.FailOnDiagnostics,
				termsOfService:      cfg.TermsOfService,
				contact:             cfg.Contact,
				license:             cfg.License,
				logger:              logger,
			}, nil
		}, di.AutoGroupIgnoreType[web.Handler](web.HandlersGroupName)),
		di.Invoke(func(
			app *fiber.App,
			middleware *Middleware,
			logger *zap.Logger,
			customizers []Customizer,
			preCustomizers []PreRun,
			postCustomizers []PostRun,
		) {
			routes := InspectFiberRoutes(app, logger)
			extraModels := combineExtraModelTypes(
				middleware.modelRegistry,
				middleware.extraModels,
			)
			hooks := composeSwaggerCustomizeHooks(middleware.hook, customizers, preCustomizers, postCustomizers)
			buildOpts := OpenAPIBuildOptions{
				SecuritySchemes:     middleware.securitySchemes,
				DefaultSecurity:     middleware.defaultSecurity,
				TagDescriptions:     middleware.tagDescriptions,
				PackageTagTransform: middleware.packageTagTransform,
				IncludeDiagnostics:  middleware.includeDiagnostics,
				DiagnosticsSeverity: middleware.diagnosticsSeverity,
				FailOnDiagnostics:   middleware.failOnDiagnostics,
				TermsOfService:      middleware.termsOfService,
				Contact:             middleware.contact,
				License:             middleware.license,
				PreHook:             hooks.pre,
				Hook:                hooks.run,
				PostHook:            hooks.post,
				ExtraModels:         extraModels,
			}
			middleware.spec = BuildOpenAPISpecWithRegistryAndOptions(routes, middleware.config, middleware.registry, buildOpts)
			middleware.versionedSpecs = middleware.versionedSpecs[:0]
			for _, doc := range middleware.versionedDocs {
				filtered := filterRoutesByPrefix(routes, doc.Prefix)
				cfg := middleware.config
				if strings.TrimSpace(doc.Name) != "" {
					cfg.Name = doc.Name
				}
				spec := BuildOpenAPISpecWithRegistryAndOptions(filtered, cfg, middleware.registry, buildOpts)
				middleware.versionedSpecs = append(middleware.versionedSpecs, mountedSpec{
					path: doc.Path,
					spec: spec,
				})
				logger.Info("auto-swagger: generated versioned OpenAPI spec",
					zap.Int("routes", len(filtered)),
					zap.String("prefix", doc.Prefix),
					zap.String("ui_path", doc.Path),
					zap.String("spec_path", doc.Path+"/swagger.json"),
				)
			}

			activeRegistry := middleware.registry
			if activeRegistry != nil {
				for key, meta := range activeRegistry.AllRoutes() {
					logger.Debug("registered metadata",
						zap.String("key", key),
						zap.String("operationId", meta.OperationID),
						zap.Strings("tags", meta.Tags),
					)
				}
			}

			logger.Info("auto-swagger: generated OpenAPI spec",
				zap.Int("routes", len(routes)),
				zap.String("ui_path", middleware.path),
				zap.String("spec_path", middleware.path+"/swagger.json"),
			)
			middleware.emitSpecFiles()
			middleware.Handle(web.NewRouterWithRegistry(app, middleware.registry))
		}, web.Priority(web.Latest), di.Params(
			``,
			``,
			``,
			di.Group(CustomizersGroupName),
			di.Group(PreCustomizersGroupName),
			di.Group(PostCustomizersGroupName),
		)),
	)
}

func (m *Middleware) emitSpecFiles() {
	if m == nil || len(m.emitFiles) == 0 {
		return
	}
	if !emitFilesEnabledByTag {
		return
	}
	if !meta.IsDevelopment() {
		return
	}

	for _, file := range m.emitFiles {
		path := strings.TrimSpace(file)
		if path == "" {
			continue
		}
		if m.spec != nil {
			if err := emitOpenAPIFile(path, m.spec); err != nil {
				m.logger.Warn("auto-swagger: failed to emit spec file",
					zap.String("path", path),
					zap.Error(err),
				)
			} else {
				m.logger.Info("auto-swagger: emitted OpenAPI spec file",
					zap.String("path", path),
				)
			}
		}
		for _, mounted := range m.versionedSpecs {
			target, ok := deriveVersionedEmitPath(path, m.path, mounted.path)
			if !ok {
				m.logger.Warn("auto-swagger: skipped versioned spec emit path",
					zap.String("base_emit", path),
					zap.String("docs_path", m.path),
					zap.String("versioned_docs_path", mounted.path),
				)
				continue
			}
			if err := emitOpenAPIFile(target, mounted.spec); err != nil {
				m.logger.Warn("auto-swagger: failed to emit versioned spec file",
					zap.String("path", target),
					zap.Error(err),
				)
				continue
			}
			m.logger.Info("auto-swagger: emitted versioned OpenAPI spec file",
				zap.String("path", target),
				zap.String("versioned_docs_path", mounted.path),
			)
		}
	}
}

func deriveVersionedEmitPath(baseEmitFile, docsPath, versionedDocsPath string) (string, bool) {
	baseEmitFile = filepath.Clean(strings.TrimSpace(baseEmitFile))
	if baseEmitFile == "." || baseEmitFile == "" {
		return "", false
	}

	docsPath = normalizeDocsPath(docsPath)
	versionedDocsPath = normalizeDocsPath(versionedDocsPath)
	if docsPath == "" || versionedDocsPath == "" {
		return "", false
	}
	if docsPath == versionedDocsPath {
		return baseEmitFile, true
	}
	if !strings.HasPrefix(versionedDocsPath, docsPath+"/") {
		return "", false
	}

	relative := strings.TrimPrefix(versionedDocsPath, docsPath)
	relative = strings.TrimPrefix(relative, "/")
	if strings.TrimSpace(relative) == "" {
		return baseEmitFile, true
	}

	baseDir := filepath.Dir(baseEmitFile)
	fileName := filepath.Base(baseEmitFile)
	return filepath.Join(baseDir, filepath.FromSlash(relative), fileName), true
}

func emitOpenAPIFile(path string, spec *OpenAPISpec) error {
	if spec == nil {
		return fmt.Errorf("openapi spec is nil")
	}

	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || path == "" {
		return fmt.Errorf("invalid emit path")
	}

	data, err := marshalOpenAPIForPath(path, spec)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return nil
}

func marshalOpenAPIForPath(path string, spec *OpenAPISpec) ([]byte, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(path)))
	switch ext {
	case ".yaml", ".yml":
		body, err := yaml.Marshal(spec)
		if err != nil {
			return nil, err
		}
		return append([]byte(buildEmitBannerComment(spec)), body...), nil
	case ".json", "":
		return json.MarshalIndent(spec, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported emit file extension %q (use .json, .yaml, or .yml)", ext)
	}
}

func buildEmitBannerComment(spec *OpenAPISpec) string {
	return strings.Join([]string{
		"# Generated by Ultrastructure AutoSwag v" + emitVersion(),
		"# Do not edit manually.",
		"# Project: " + emitProjectName(spec),
		"# OpenAPI spec version: " + emitOpenAPIVersion(spec),
		"",
	}, "\n")
}

func emitVersion() string {
	v := strings.TrimSpace(meta.Version)
	if v == "" {
		return "unknown"
	}
	return v
}

func emitProjectName(spec *OpenAPISpec) string {
	if spec == nil {
		return "unknown"
	}
	name := strings.TrimSpace(spec.Info.Title)
	if name == "" {
		return "unknown"
	}
	return name
}

func emitOpenAPIVersion(spec *OpenAPISpec) string {
	if spec == nil {
		return "unknown"
	}
	v := strings.TrimSpace(spec.OpenAPI)
	if v == "" {
		return "unknown"
	}
	return v
}

// Handle registers the auto-swagger routes.
func (m *Middleware) Handle(r web.Router) {
	m.registerSpecEndpoints(r, m.path, m.spec)
	for _, mounted := range m.versionedSpecs {
		m.registerSpecEndpoints(r, mounted.path, mounted.spec)
	}

	m.logger.Debug("auto-swagger endpoints registered",
		zap.String("ui", m.path),
		zap.String("spec", m.path+"/swagger.json"),
	)
}

func (m *Middleware) registerSpecEndpoints(r web.Router, path string, spec *OpenAPISpec) {
	r.Get(path+"/swagger.json", func(c fiber.Ctx) error {
		specJSON, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate OpenAPI spec",
			})
		}

		c.Set("Content-Type", web.ContentTypeApplicationJSON)
		return c.Send(specJSON)
	})

	r.Get(path, func(c fiber.Ctx) error {
		html := m.generateSwaggerUI(path + "/swagger.json")
		c.Set("Content-Type", web.ContentTypeTextHTML)
		return c.SendString(html)
	})
}

func (m *Middleware) generateSwaggerUI(specURL string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + m.config.Name + ` - API Documentation</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.31.1/swagger-ui.css">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
        .topbar {
            display: none;
        }
        .swagger-ui .info {
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.31.1/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.31.1/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "` + specURL + `",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                docExpansion: "list",
                filter: true,
                showRequestHeaders: true,
                tryItOutEnabled: true
            });
            window.ui = ui;
        };
    </script>
</body>
</html>`
}

func filterRoutesByPrefix(routes []RouteInfo, prefix string) []RouteInfo {
	prefix = normalizeRoutePrefix(prefix)
	if prefix == "" {
		return append([]RouteInfo(nil), routes...)
	}

	out := make([]RouteInfo, 0, len(routes))
	for _, route := range routes {
		path := strings.TrimSpace(route.Path)
		if path == "" {
			path = "/"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			out = append(out, route)
		}
	}
	return out
}
