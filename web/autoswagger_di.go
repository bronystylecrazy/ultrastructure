package web

import (
	"encoding/json"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

// AutoSwaggerMiddleware provides runtime OpenAPI spec generation
type AutoSwaggerMiddleware struct {
	config          Config
	path            string
	securitySchemes map[string]interface{}
	defaultSecurity []SecurityRequirement
	tagDescriptions map[string]string
	termsOfService  string
	contact         *AutoSwaggerContact
	license         *AutoSwaggerLicense
	spec            *AutoSwaggerSpec
	logger          *zap.Logger
}

// UseAutoSwagger enables automatic Swagger documentation generation
// This inspects all registered routes at runtime and generates an OpenAPI spec
func UseAutoSwagger(opts ...SwaggerOption) di.Node {
	return di.Options(
		// Provide the auto-swagger middleware
		di.Provide(func(config Config, logger *zap.Logger) (*AutoSwaggerMiddleware, error) {
			cfg := swaggerOptions{path: "/docs"}
			for _, opt := range opts {
				if opt != nil {
					opt(&cfg)
				}
			}

			return &AutoSwaggerMiddleware{
				config:          config,
				path:            cfg.path,
				securitySchemes: cfg.securitySchemes,
				defaultSecurity: append([]SecurityRequirement(nil), cfg.defaultSecurity...),
				tagDescriptions: cfg.tagDescriptions,
				termsOfService:  cfg.termsOfService,
				contact:         cfg.contact,
				license:         cfg.license,
				logger:          logger,
			}, nil
		}, IgnoreAutoGroupHandlers()),

		// Hook into app lifecycle to inspect routes after all handlers are registered
		di.Invoke(func(app *fiber.App, middleware *AutoSwaggerMiddleware, logger *zap.Logger) {
			// This runs after all handlers are set up
			// We inspect the Fiber app's routes and build the OpenAPI spec
			routes := InspectFiberRoutes(app, logger)
			middleware.spec = BuildOpenAPISpecWithOptions(routes, middleware.config, OpenAPIBuildOptions{
				SecuritySchemes: middleware.securitySchemes,
				DefaultSecurity: middleware.defaultSecurity,
				TagDescriptions: middleware.tagDescriptions,
				TermsOfService:  middleware.termsOfService,
				Contact:         middleware.contact,
				License:         middleware.license,
			})

			// Debug: log all registered metadata routes
			logger.Debug("metadata registry routes")
			for key, meta := range GetGlobalRegistry().AllRoutes() {
				logger.Debug("registered metadata",
					zap.String("key", key),
					zap.String("operationId", meta.OperationID),
					zap.Strings("tags", meta.Tags),
				)
			}

			logger.Info("auto-swagger: generated OpenAPI spec",
				zap.Int("routes", len(routes)),
				zap.String("ui_path", middleware.path),
				zap.String("spec_path", middleware.path+"/swagger.json"),
			)

			// Register the swagger endpoints
			middleware.Handle(app)
		}, Priority(Latest)),
	)
}

// Handle registers the auto-swagger routes
func (m *AutoSwaggerMiddleware) Handle(r fiber.Router) {
	// Serve the OpenAPI spec as JSON
	r.Get(m.path+"/swagger.json", func(c fiber.Ctx) error {
		specJSON, err := json.MarshalIndent(m.spec, "", "  ")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate OpenAPI spec",
			})
		}

		c.Set("Content-Type", "application/json")
		return c.Send(specJSON)
	})

	// Serve Swagger UI HTML
	r.Get(m.path, func(c fiber.Ctx) error {
		html := m.generateSwaggerUI()
		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	})

	m.logger.Debug("auto-swagger endpoints registered",
		zap.String("ui", m.path),
		zap.String("spec", m.path+"/swagger.json"),
	)
}

// generateSwaggerUI creates a Swagger UI HTML page
func (m *AutoSwaggerMiddleware) generateSwaggerUI() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + m.config.Name + ` - API Documentation</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
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
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "` + m.path + `/swagger.json",
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
