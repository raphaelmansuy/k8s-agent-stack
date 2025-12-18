// Package validation provides input validation and sanitization for security.
package validation

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// Patterns for validation
	slugPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
	uuidPattern  = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	alphanumeric = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	// SQL injection patterns (case insensitive)
	sqlPatterns = regexp.MustCompile(`(?i)(SELECT|INSERT|UPDATE|DELETE|DROP|UNION|ALTER|CREATE|TRUNCATE|EXEC|EXECUTE|XP_|SP_|0X|WAITFOR|BENCHMARK|SLEEP)[\s\(]`)

	// XSS patterns
	xssPatterns = regexp.MustCompile(`(?i)(<script|javascript:|on\w+\s*=|<iframe|<object|<embed|<link|<style|<img[^>]+onerror)`)

	// Path traversal patterns
	pathTraversal = regexp.MustCompile(`(\.\.[\\/]|[\\/]\.\.|\.\.|%2e%2e|%252e%252e)`)
)

// Errors
var (
	ErrInvalidSlug          = errors.New("invalid slug format")
	ErrInvalidEmail         = errors.New("invalid email format")
	ErrInvalidUUID          = errors.New("invalid UUID format")
	ErrStringTooLong        = errors.New("string exceeds maximum length")
	ErrStringTooShort       = errors.New("string is too short")
	ErrSQLInjectionDetected = errors.New("potential SQL injection detected")
	ErrXSSDetected          = errors.New("potential XSS detected")
	ErrPathTraversal        = errors.New("path traversal detected")
	ErrInvalidCharacters    = errors.New("invalid characters in input")
	ErrEmptyInput           = errors.New("input cannot be empty")
)

// SanitizeString removes dangerous characters from a string.
func SanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")

	// Remove control characters (except newlines and tabs)
	var result strings.Builder
	result.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' || r == '\r' || !unicode.IsControl(r) {
			result.WriteRune(r)
		}
	}

	return strings.TrimSpace(result.String())
}

// SanitizeHTML removes all HTML tags from a string.
func SanitizeHTML(s string) string {
	// Simple HTML stripper - for production use bluemonday
	inTag := false
	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}

	// Decode common HTML entities
	text := result.String()
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return text
}

// ValidateSlug checks if a string is a valid URL slug.
func ValidateSlug(s string) error {
	if len(s) < 2 {
		return ErrStringTooShort
	}
	if len(s) > 63 {
		return ErrStringTooLong
	}
	if !slugPattern.MatchString(s) {
		return ErrInvalidSlug
	}
	return nil
}

// ValidateEmail checks if a string is a valid email address.
func ValidateEmail(s string) error {
	if s == "" {
		return ErrEmptyInput
	}
	if len(s) > 254 {
		return ErrStringTooLong
	}
	_, err := mail.ParseAddress(s)
	if err != nil {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateUUID checks if a string is a valid UUID.
func ValidateUUID(s string) error {
	if !uuidPattern.MatchString(strings.ToLower(s)) {
		return ErrInvalidUUID
	}
	return nil
}

// ValidateLength checks if a string is within length bounds.
func ValidateLength(s string, min, max int) error {
	length := utf8.RuneCountInString(s)
	if length < min {
		return ErrStringTooShort
	}
	if length > max {
		return ErrStringTooLong
	}
	return nil
}

// DetectSQLInjection checks for SQL injection patterns.
func DetectSQLInjection(s string) bool {
	return sqlPatterns.MatchString(s)
}

// DetectXSS checks for XSS patterns.
func DetectXSS(s string) bool {
	return xssPatterns.MatchString(s)
}

// DetectPathTraversal checks for path traversal patterns.
func DetectPathTraversal(s string) bool {
	return pathTraversal.MatchString(s)
}

// ValidateNoInjection checks for common injection patterns.
func ValidateNoInjection(s string) error {
	if DetectSQLInjection(s) {
		return ErrSQLInjectionDetected
	}
	if DetectXSS(s) {
		return ErrXSSDetected
	}
	if DetectPathTraversal(s) {
		return ErrPathTraversal
	}
	return nil
}

// JSONValidator validates JSON structure.
type JSONValidator struct {
	MaxDepth    int
	MaxKeyLen   int
	MaxValueLen int
	MaxArrayLen int
}

// NewJSONValidator creates a new JSON validator with sensible defaults.
func NewJSONValidator() *JSONValidator {
	return &JSONValidator{
		MaxDepth:    10,
		MaxKeyLen:   64,
		MaxValueLen: 1024 * 1024, // 1MB
		MaxArrayLen: 1000,
	}
}

// Validate checks if a JSON structure is within limits.
func (v *JSONValidator) Validate(data map[string]any) error {
	return v.validateMap(data, 0)
}

func (v *JSONValidator) validateMap(data map[string]any, depth int) error {
	if depth > v.MaxDepth {
		return fmt.Errorf("JSON depth exceeds maximum of %d", v.MaxDepth)
	}

	for key, value := range data {
		if len(key) > v.MaxKeyLen {
			return fmt.Errorf("key '%s...' exceeds maximum length of %d", key[:20], v.MaxKeyLen)
		}

		switch val := value.(type) {
		case string:
			if len(val) > v.MaxValueLen {
				return fmt.Errorf("value for key '%s' exceeds maximum length", key)
			}
			// Check for injection in string values
			if err := ValidateNoInjection(val); err != nil {
				return fmt.Errorf("invalid value for key '%s': %w", key, err)
			}
		case map[string]any:
			if err := v.validateMap(val, depth+1); err != nil {
				return err
			}
		case []any:
			if len(val) > v.MaxArrayLen {
				return fmt.Errorf("array for key '%s' exceeds maximum length of %d", key, v.MaxArrayLen)
			}
			for i, item := range val {
				if m, ok := item.(map[string]any); ok {
					if err := v.validateMap(m, depth+1); err != nil {
						return err
					}
				}
				if s, ok := item.(string); ok {
					if len(s) > v.MaxValueLen {
						return fmt.Errorf("array item %d exceeds maximum length", i)
					}
				}
			}
		}
	}

	return nil
}

// FieldValidator validates struct fields.
type FieldValidator struct {
	errors []string
}

// NewFieldValidator creates a new field validator.
func NewFieldValidator() *FieldValidator {
	return &FieldValidator{
		errors: make([]string, 0),
	}
}

// Required checks if a string field is not empty.
func (v *FieldValidator) Required(field, name string) *FieldValidator {
	if strings.TrimSpace(field) == "" {
		v.errors = append(v.errors, fmt.Sprintf("%s is required", name))
	}
	return v
}

// MinLength checks minimum length.
func (v *FieldValidator) MinLength(field, name string, min int) *FieldValidator {
	if utf8.RuneCountInString(field) < min {
		v.errors = append(v.errors, fmt.Sprintf("%s must be at least %d characters", name, min))
	}
	return v
}

// MaxLength checks maximum length.
func (v *FieldValidator) MaxLength(field, name string, max int) *FieldValidator {
	if utf8.RuneCountInString(field) > max {
		v.errors = append(v.errors, fmt.Sprintf("%s must be at most %d characters", name, max))
	}
	return v
}

// Email validates email format.
func (v *FieldValidator) Email(field, name string) *FieldValidator {
	if err := ValidateEmail(field); err != nil {
		v.errors = append(v.errors, fmt.Sprintf("%s must be a valid email address", name))
	}
	return v
}

// Slug validates slug format.
func (v *FieldValidator) Slug(field, name string) *FieldValidator {
	if err := ValidateSlug(field); err != nil {
		v.errors = append(v.errors, fmt.Sprintf("%s must be a valid slug (lowercase letters, numbers, and hyphens)", name))
	}
	return v
}

// NoInjection checks for injection patterns.
func (v *FieldValidator) NoInjection(field, name string) *FieldValidator {
	if err := ValidateNoInjection(field); err != nil {
		v.errors = append(v.errors, fmt.Sprintf("%s contains invalid characters", name))
	}
	return v
}

// Custom adds a custom validation.
func (v *FieldValidator) Custom(condition bool, message string) *FieldValidator {
	if !condition {
		v.errors = append(v.errors, message)
	}
	return v
}

// HasErrors returns true if there are validation errors.
func (v *FieldValidator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all validation errors.
func (v *FieldValidator) Errors() []string {
	return v.errors
}

// Error returns the first error or nil.
func (v *FieldValidator) Error() error {
	if len(v.errors) == 0 {
		return nil
	}
	return fmt.Errorf("validation failed: %s", strings.Join(v.errors, "; "))
}
