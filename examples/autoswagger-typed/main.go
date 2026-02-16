package main

import (
	"reflect"
	"strings"
	"time"

	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Domain models with JSON tags
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
	Name  string `json:"name" description:"User's full name"`
	Email string `json:"email" description:"User's email address"`
}

type UpdateUserRequest struct {
	Name  string `json:"name,omitempty" description:"User's full name (optional)"`
	Email string `json:"email,omitempty" description:"User's email address (optional)"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type HealthStatus struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime"`
	Version string `json:"version"`
}

// UserHandler with fluent API metadata
type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) Handle(router web.Router) {
	// Create a group with inherited tags - all routes in this group will have "Users" tag
	users := router.Group("/users").Tags("Users")

	// GET /users/:id - with fluent API metadata
	users.Get("/:id", h.GetUser).
		Name("GetUserByID").
		Summary("Get a user by ID").
		Description("Retrieves detailed information about a specific user").
		HeaderRequired("X-Tenant-ID", "", "Tenant identifier").
		Ok(User{}, "User found").
		NotFound(ErrorResponse{}, "User not found")

	// POST /users - Create user with typed request/response
	users.Post("", h.CreateUser).
		Name("CreateUser").
		Summary("Create a new user").
		Description("Creates a new user account with the provided information").
		Body(CreateUserRequest{}).
		Create(User{}, "User created").
		BadRequest(ErrorResponse{}, "Invalid request body")

	// PUT /users/:id - Update user
	users.Put("/:id", h.UpdateUser).
		Name("UpdateUser").
		Summary("Update a user").
		Description("Updates an existing user's information").
		Body(UpdateUserRequest{}).
		Ok(User{}, "User updated").
		NotFound(ErrorResponse{}, "User not found").
		BadRequest(ErrorResponse{}, "Invalid request body")

	// DELETE /users/:id
	users.Delete("/:id", h.DeleteUser).
		Name("DeleteUser").
		Summary("Delete a user").
		NoContent(204).
		NotFound(ErrorResponse{}, "User not found")

	// GET /health - Simple endpoint with different tags
	router.Get("/health", h.Health).
		Name("HealthCheck").
		Tags("System").
		Summary("Health check endpoint").
		Ok(HealthStatus{}, "Service is healthy")
}

func (h *UserHandler) GetUser(c fiber.Ctx) error {
	id := c.Params("id")
	user := User{
		ID:        id,
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}
	return c.JSON(user)
}

func (h *UserHandler) CreateUser(c fiber.Ctx) error {
	var req CreateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_INPUT",
			Message: err.Error(),
		})
	}

	user := User{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}
	return c.Status(201).JSON(user)
}

func (h *UserHandler) UpdateUser(c fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_INPUT",
			Message: err.Error(),
		})
	}

	user := User{
		ID:        id,
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}
	return c.JSON(user)
}

func (h *UserHandler) DeleteUser(c fiber.Ctx) error {
	return c.SendStatus(204)
}

func (h *UserHandler) Health(c fiber.Ctx) error {
	return c.JSON(HealthStatus{
		Status:  "healthy",
		Uptime:  "24h",
		Version: "1.0.0",
	})
}

type CustomTest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type UsersError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// UsersSwaggerCustomizer demonstrates DI-provided per-route swagger customization.
type UsersSwaggerCustomizer struct{}

func NewUsersSwaggerCustomizer() *UsersSwaggerCustomizer {
	return &UsersSwaggerCustomizer{}
}

func (c *UsersSwaggerCustomizer) CustomizeSwagger(ctx *web.SwaggerContext) {
	if ctx.Route.Path != "/users/:id" || strings.ToUpper(ctx.Route.Method) != "GET" {
		return
	}

	if ctx.Metadata.Responses == nil {
		ctx.Metadata.Responses = map[int]web.ResponseMetadata{}
	}

	ctx.Metadata.Responses[400] = web.ResponseMetadata{
		Type:        reflect.TypeOf(UsersError{}),
		ContentType: "application/json",
		Description: "Users endpoint error",
	}
	ctx.Metadata.Responses[500] = web.ResponseMetadata{
		Type:        reflect.TypeOf(UsersError{}),
		ContentType: "application/json",
		Description: "Users endpoint internal error",
	}

	// Add a named schema from the customizer context.
	if ctx.Models != nil {
		ctx.Models.AddNamed("CustomTest", CustomTest{})
	}
}

func main() {
	us.New(
		web.Init(),
		di.Provide(NewUserHandler),
		di.Provide(NewUsersSwaggerCustomizer),
		web.UseAutoSwagger(
			web.WithSwaggerCustomize(func(ctx *web.SwaggerContext) {
				// Option hook and DI customizer are composable.
				if len(ctx.Metadata.Tags) > 0 && ctx.Metadata.OperationID != "" {
					ctx.Metadata.OperationID = ctx.Metadata.Tags[0] + "__" + ctx.Metadata.OperationID
				}
			}),
		),
	).Run()
}
