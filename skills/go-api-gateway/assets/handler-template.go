// Package handlers provides HTTP handler templates for AgentStack API
// Copy and customize this template for new resource handlers
package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()

// ResourceHandler handles CRUD operations for a resource
type ResourceHandler struct {
	service ResourceService
}

// ResourceService defines the business logic interface
type ResourceService interface {
	Create(ctx context.Context, projectID string, input CreateInput) (*Resource, error)
	Get(ctx context.Context, projectID, id string) (*Resource, error)
	List(ctx context.Context, projectID, cursor string, limit int) ([]*Resource, string, error)
	Update(ctx context.Context, projectID, id string, input UpdateInput) (*Resource, error)
	Delete(ctx context.Context, projectID, id string) error
}

// NewResourceHandler creates a new handler instance
func NewResourceHandler(svc ResourceService) *ResourceHandler {
	return &ResourceHandler{service: svc}
}

// Create handles POST /v1/resources
func (h *ResourceHandler) Create(c *fiber.Ctx) error {
	var req CreateResourceRequest
	if err := c.BodyParser(&req); err != nil {
		return NewError(c, fiber.StatusBadRequest, "Invalid Request", "Request body could not be parsed")
	}

	if err := validate.Struct(req); err != nil {
		return ValidationError(c, err)
	}

	projectID := c.Locals("projectID").(string)

	resource, err := h.service.Create(c.Context(), projectID, CreateInput{
		Name: req.Name,
		// ... map other fields
	})
	if err != nil {
		return handleDomainError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toResponse(resource))
}

// Get handles GET /v1/resources/:id
func (h *ResourceHandler) Get(c *fiber.Ctx) error {
	projectID := c.Locals("projectID").(string)
	id := c.Params("id")

	resource, err := h.service.Get(c.Context(), projectID, id)
	if err != nil {
		return handleDomainError(c, err)
	}

	return c.JSON(toResponse(resource))
}

// List handles GET /v1/resources
func (h *ResourceHandler) List(c *fiber.Ctx) error {
	projectID := c.Locals("projectID").(string)
	cursor := c.Query("cursor")
	limit := c.QueryInt("limit", 20)

	if limit > 100 {
		limit = 100
	}

	resources, nextCursor, err := h.service.List(c.Context(), projectID, cursor, limit)
	if err != nil {
		return handleDomainError(c, err)
	}

	return c.JSON(ListResponse{
		Data:       toResponses(resources),
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	})
}

// Update handles PATCH /v1/resources/:id
func (h *ResourceHandler) Update(c *fiber.Ctx) error {
	var req UpdateResourceRequest
	if err := c.BodyParser(&req); err != nil {
		return NewError(c, fiber.StatusBadRequest, "Invalid Request", "Request body could not be parsed")
	}

	projectID := c.Locals("projectID").(string)
	id := c.Params("id")

	resource, err := h.service.Update(c.Context(), projectID, id, UpdateInput{
		Name: req.Name,
		// ... map other fields
	})
	if err != nil {
		return handleDomainError(c, err)
	}

	return c.JSON(toResponse(resource))
}

// Delete handles DELETE /v1/resources/:id
func (h *ResourceHandler) Delete(c *fiber.Ctx) error {
	projectID := c.Locals("projectID").(string)
	id := c.Params("id")

	if err := h.service.Delete(c.Context(), projectID, id); err != nil {
		return handleDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// --- Request/Response DTOs ---

type CreateResourceRequest struct {
	Name string `json:"name" validate:"required,min=3,max=64"`
	// Add more fields as needed
}

type UpdateResourceRequest struct {
	Name *string `json:"name" validate:"omitempty,min=3,max=64"`
	// Use pointers for optional fields
}

type ResourceResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListResponse struct {
	Data       []ResourceResponse `json:"data"`
	NextCursor string             `json:"next_cursor,omitempty"`
	HasMore    bool               `json:"has_more"`
}

// --- Error Handling ---

type ErrorResponse struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail"`
	Instance string       `json:"instance,omitempty"`
	TraceID  string       `json:"trace_id,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func NewError(c *fiber.Ctx, status int, title, detail string) error {
	traceID, _ := c.Locals("traceID").(string)
	return c.Status(status).JSON(ErrorResponse{
		Type:     fmt.Sprintf("https://api.agentstack.io/errors/%s", strings.ToLower(strings.ReplaceAll(title, " ", "-"))),
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: c.Path(),
		TraceID:  traceID,
	})
}

func ValidationError(c *fiber.Ctx, err error) error {
	var fieldErrors []FieldError
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			fieldErrors = append(fieldErrors, FieldError{
				Field:   strings.ToLower(e.Field()),
				Message: validationMessage(e),
			})
		}
	}

	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Type:     "https://api.agentstack.io/errors/validation",
		Title:    "Validation Error",
		Status:   400,
		Detail:   "One or more fields failed validation",
		Instance: c.Path(),
		Errors:   fieldErrors,
	})
}

func handleDomainError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return NewError(c, fiber.StatusNotFound, "Not Found", "Resource not found")
	case errors.Is(err, ErrAlreadyExists):
		return NewError(c, fiber.StatusConflict, "Conflict", "Resource already exists")
	case errors.Is(err, ErrForbidden):
		return NewError(c, fiber.StatusForbidden, "Forbidden", "Access denied")
	default:
		// Log unexpected error
		return NewError(c, fiber.StatusInternalServerError, "Internal Error", "An unexpected error occurred")
	}
}

func validationMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "min":
		return fmt.Sprintf("Must be at least %s characters", e.Param())
	case "max":
		return fmt.Sprintf("Must be at most %s characters", e.Param())
	case "email":
		return "Must be a valid email address"
	case "oneof":
		return fmt.Sprintf("Must be one of: %s", e.Param())
	default:
		return "Invalid value"
	}
}

// --- Helpers ---

func toResponse(r *Resource) ResourceResponse {
	return ResourceResponse{
		ID:        r.ID,
		Name:      r.Name,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func toResponses(resources []*Resource) []ResourceResponse {
	result := make([]ResourceResponse, len(resources))
	for i, r := range resources {
		result[i] = toResponse(r)
	}
	return result
}
