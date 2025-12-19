/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package validation

import (
	"errors"
	"testing"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes null bytes",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "preserves newlines and tabs",
			input:    "hello\nworld\ttab",
			expected: "hello\nworld\ttab",
		},
		{
			name:     "removes control characters",
			input:    "hello\x01\x02world",
			expected: "helloworld",
		},
		{
			name:     "trims whitespace",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes script tags",
			input:    "<script>alert('xss')</script>hello",
			expected: "alert('xss')hello",
		},
		{
			name:     "removes nested tags",
			input:    "<div><p>hello</p></div>",
			expected: "hello",
		},
		{
			name:     "decodes HTML entities",
			input:    "&lt;script&gt;",
			expected: "<script>",
		},
		{
			name:     "handles plain text",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "valid slug",
			input:   "hello-world",
			wantErr: nil,
		},
		{
			name:    "valid slug with numbers",
			input:   "hello-123-world",
			wantErr: nil,
		},
		{
			name:    "too short",
			input:   "a",
			wantErr: ErrStringTooShort,
		},
		{
			name:    "invalid characters",
			input:   "Hello_World",
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "starts with hyphen",
			input:   "-hello",
			wantErr: ErrInvalidSlug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlug(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateSlug(%q) = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "valid email",
			input:   "user@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with subdomain",
			input:   "user@mail.example.com",
			wantErr: nil,
		},
		{
			name:    "empty email",
			input:   "",
			wantErr: ErrEmptyInput,
		},
		{
			name:    "invalid email no domain",
			input:   "user@",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "invalid email no at",
			input:   "userexample.com",
			wantErr: ErrInvalidEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "valid lowercase UUID",
			input:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: nil,
		},
		{
			name:    "valid uppercase UUID",
			input:   "550E8400-E29B-41D4-A716-446655440000",
			wantErr: nil,
		},
		{
			name:    "invalid UUID format",
			input:   "not-a-uuid",
			wantErr: ErrInvalidUUID,
		},
		{
			name:    "invalid UUID too short",
			input:   "550e8400-e29b-41d4-a716",
			wantErr: ErrInvalidUUID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUID(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateUUID(%q) = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestDetectSQLInjection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "simple SELECT",
			input:    "SELECT * FROM users",
			expected: true,
		},
		{
			name:     "UNION attack",
			input:    "1 UNION SELECT password FROM users",
			expected: true,
		},
		{
			name:     "DROP TABLE",
			input:    "'; DROP TABLE users; --",
			expected: true,
		},
		{
			name:     "safe input",
			input:    "John Doe",
			expected: false,
		},
		{
			name:     "safe input with select word",
			input:    "I selected the best option",
			expected: false, // word boundaries matter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectSQLInjection(tt.input)
			if result != tt.expected {
				t.Errorf("DetectSQLInjection(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectXSS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "script tag",
			input:    "<script>alert('xss')</script>",
			expected: true,
		},
		{
			name:     "javascript protocol",
			input:    "javascript:alert('xss')",
			expected: true,
		},
		{
			name:     "onerror handler",
			input:    "<img src=x onerror=alert('xss')>",
			expected: true,
		},
		{
			name:     "iframe tag",
			input:    "<iframe src='evil.com'>",
			expected: true,
		},
		{
			name:     "safe input",
			input:    "Hello World",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectXSS(tt.input)
			if result != tt.expected {
				t.Errorf("DetectXSS(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectPathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "unix path traversal",
			input:    "../../../etc/passwd",
			expected: true,
		},
		{
			name:     "windows path traversal",
			input:    "..\\..\\windows\\system32",
			expected: true,
		},
		{
			name:     "url encoded",
			input:    "%2e%2e%2f%2e%2e%2f",
			expected: true,
		},
		{
			name:     "safe path",
			input:    "/var/www/html/index.html",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectPathTraversal(tt.input)
			if result != tt.expected {
				t.Errorf("DetectPathTraversal(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFieldValidator(t *testing.T) {
	t.Run("required field", func(t *testing.T) {
		v := NewFieldValidator().Required("", "name")
		if !v.HasErrors() {
			t.Error("expected error for empty required field")
		}
	})

	t.Run("min length", func(t *testing.T) {
		v := NewFieldValidator().MinLength("ab", "name", 3)
		if !v.HasErrors() {
			t.Error("expected error for field shorter than min")
		}
	})

	t.Run("max length", func(t *testing.T) {
		v := NewFieldValidator().MaxLength("hello", "name", 3)
		if !v.HasErrors() {
			t.Error("expected error for field longer than max")
		}
	})

	t.Run("valid chain", func(t *testing.T) {
		v := NewFieldValidator().
			Required("hello", "name").
			MinLength("hello", "name", 3).
			MaxLength("hello", "name", 10)
		if v.HasErrors() {
			t.Errorf("unexpected errors: %v", v.Errors())
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		v := NewFieldValidator().
			Required("", "name").
			Required("", "email")
		if len(v.Errors()) != 2 {
			t.Errorf("expected 2 errors, got %d", len(v.Errors()))
		}
	})
}

func TestJSONValidator(t *testing.T) {
	v := NewJSONValidator()

	t.Run("valid JSON", func(t *testing.T) {
		data := map[string]any{
			"name":  "test",
			"value": 123,
		}
		if err := v.Validate(data); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("key too long", func(t *testing.T) {
		longKey := ""
		for range 100 {
			longKey += "a"
		}
		data := map[string]any{
			longKey: "value",
		}
		if err := v.Validate(data); err == nil {
			t.Error("expected error for key too long")
		}
	})

	t.Run("nested depth exceeded", func(t *testing.T) {
		// Create a deeply nested structure that exceeds MaxDepth of 10
		data := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"level4": map[string]any{
							"level5": map[string]any{
								"level6": map[string]any{
									"level7": map[string]any{
										"level8": map[string]any{
											"level9": map[string]any{
												"level10": map[string]any{
													"level11": map[string]any{
														"level12": "too deep",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		if err := v.Validate(data); err == nil {
			t.Error("expected error for depth exceeded")
		}
	})
}
