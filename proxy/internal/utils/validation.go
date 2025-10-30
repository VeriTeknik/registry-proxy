package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	// validate is the singleton validator instance
	validate *validator.Validate

	// serverIDRegex matches valid server IDs (UUIDs and qualified names)
	serverIDRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,254}$`)

	// uuidRegex matches UUIDs
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func init() {
	validate = validator.New()
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Limit  int `validate:"min=1,max=1000"`
	Offset int `validate:"min=0"`
}

// ServerFilterRequest represents filter parameters for server queries
type ServerFilterRequest struct {
	Search        string   `validate:"omitempty,max=200"`
	Category      string   `validate:"omitempty,max=100"`
	MinRating     float64  `validate:"omitempty,min=0,max=5"`
	MinInstalls   int      `validate:"omitempty,min=0"`
	RegistryTypes []string `validate:"omitempty,dive,oneof=npm pypi oci mcpb nuget remote"`
	Tags          []string `validate:"omitempty,dive,max=50"`
	HasTransport  []string `validate:"omitempty,dive,oneof=stdio sse http"`
}

// RatingRequest represents a server rating submission
type RatingRequest struct {
	Rating  float64 `json:"rating" validate:"required,min=1,max=5"`
	Comment string  `json:"comment" validate:"omitempty,max=1000"`
}

// InstallRequest represents a server installation event
type InstallRequest struct {
	Version  string `json:"version" validate:"omitempty,max=50"`
	Platform string `json:"platform" validate:"omitempty,max=50"`
}

// ValidateStruct validates a struct using validator.v10
func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return formatValidationErrors(validationErrors)
		}
		return err
	}
	return nil
}

// formatValidationErrors converts validator errors to user-friendly messages
func formatValidationErrors(errs validator.ValidationErrors) error {
	var messages []string
	for _, err := range errs {
		messages = append(messages, formatFieldError(err))
	}
	return fmt.Errorf("validation failed: %s", strings.Join(messages, "; "))
}

// formatFieldError formats a single field validation error
func formatFieldError(err validator.FieldError) string {
	field := err.Field()
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, err.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, err.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// ValidateServerID validates a server ID for safety.
// Prevents path traversal and injection attacks.
func ValidateServerID(serverID string) error {
	if serverID == "" {
		return fmt.Errorf("server ID cannot be empty")
	}

	if len(serverID) > 255 {
		return fmt.Errorf("server ID too long")
	}

	// Check for path traversal attempts
	if strings.Contains(serverID, "..") || strings.Contains(serverID, "/") || strings.Contains(serverID, "\\") {
		return fmt.Errorf("invalid server ID format")
	}

	// Must match either UUID or qualified name format
	if !serverIDRegex.MatchString(serverID) && !uuidRegex.MatchString(strings.ToLower(serverID)) {
		return fmt.Errorf("invalid server ID format")
	}

	return nil
}

// ValidateSortParameter validates that a sort parameter is in the allowed whitelist
func ValidateSortParameter(sort string, validSorts []string) error {
	if sort == "" {
		return nil // Empty is okay, will use default
	}

	for _, valid := range validSorts {
		if sort == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid sort parameter: %s (valid options: %s)", sort, strings.Join(validSorts, ", "))
}
