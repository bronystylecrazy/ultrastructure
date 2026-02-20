package autoswag

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

// Middleware provides runtime OpenAPI spec generation.
type Middleware struct {
	config              web.Config
	path                string
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
