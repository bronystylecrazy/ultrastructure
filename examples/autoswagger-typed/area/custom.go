package custom

import (
	"time"

	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Domain models with JSON tags
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
	Name  string `json:"name" description:"User's full name" validate:"required"`
	Email string `json:"email" description:"User's email address" validate:"required,email"`
	Age   int    `json:"age,omitempty" description:"User age (must be less than 150)" validate:"lt=150"`
}

type UpdateUserRequest struct {
	Name  string `json:"name,omitempty" description:"User's full name (optional)" validate:"omitempty"`
	Email string `json:"email,omitempty" description:"User's email address (optional)" validate:"omitempty,email"`
	Age   *int   `json:"age,omitempty" description:"User age (must be less than 150)" validate:"omitempty,lt=150"`
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

var payloadValidator = validator.New()

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
		Age:       30,
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
	if err := validateCreateUserRequest(req); err != nil {
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
		Age:       req.Age,
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
	if err := validateUpdateUserRequest(req); err != nil {
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
		Age:       derefOrZero(req.Age),
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

func validateCreateUserRequest(req CreateUserRequest) error {
	if err := payloadValidator.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return nil
}

func validateUpdateUserRequest(req UpdateUserRequest) error {
	if err := payloadValidator.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.Name == "" && req.Email == "" && req.Age == nil {
		return fiber.NewError(fiber.StatusBadRequest, "at least one field (name, email, or age) is required")
	}
	return nil
}

func derefOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
