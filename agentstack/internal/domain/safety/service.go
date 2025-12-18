// Package safety provides content moderation and safety checking for agent interactions.
package safety

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// Service handles safety checks for agent inputs and outputs.
type Service struct {
	config    Config
	moderator Moderator
}

// Moderator defines the interface for content moderation.
type Moderator interface {
	Classify(ctx context.Context, text string) (*Classification, error)
}

// Config contains safety service configuration.
type Config struct {
	Enabled              bool       `json:"enabled"`
	BlockedPatterns      []string   `json:"blocked_patterns"`
	BlockedWords         []string   `json:"blocked_words"`
	MaxInputLength       int        `json:"max_input_length"`
	MaxOutputLength      int        `json:"max_output_length"`
	RequireModeration    bool       `json:"require_moderation"`
	FailOpen             bool       `json:"fail_open"` // If true, allow on moderation failure
	ModerationThresholds Thresholds `json:"moderation_thresholds"`
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		BlockedPatterns:   []string{},
		BlockedWords:      []string{},
		MaxInputLength:    32000,
		MaxOutputLength:   128000,
		RequireModeration: false,
		FailOpen:          true,
		ModerationThresholds: Thresholds{
			Hate:            0.8,
			Violence:        0.8,
			Sexual:          0.8,
			SelfHarm:        0.8,
			HateWithThreat:  0.6,
			ViolenceGraphic: 0.6,
		},
	}
}

// Thresholds defines moderation score thresholds.
type Thresholds struct {
	Hate            float64 `json:"hate"`
	Violence        float64 `json:"violence"`
	Sexual          float64 `json:"sexual"`
	SelfHarm        float64 `json:"self_harm"`
	HateWithThreat  float64 `json:"hate_threatening"`
	ViolenceGraphic float64 `json:"violence_graphic"`
}

// NewService creates a new safety service.
func NewService(config Config, moderator Moderator) *Service {
	return &Service{
		config:    config,
		moderator: moderator,
	}
}

// CheckInput validates user input before sending to an agent.
func (s *Service) CheckInput(ctx context.Context, input string) (*CheckResult, error) {
	if !s.config.Enabled {
		return &CheckResult{Allowed: true}, nil
	}

	result := &CheckResult{Allowed: true, Checks: []Check{}}

	// Length check
	if len(input) > s.config.MaxInputLength {
		result.Allowed = false
		result.Checks = append(result.Checks, Check{
			Name:   "length",
			Passed: false,
			Reason: fmt.Sprintf("Input exceeds maximum length of %d characters", s.config.MaxInputLength),
		})
		return result, nil
	}
	result.Checks = append(result.Checks, Check{Name: "length", Passed: true})

	// Blocked words check
	if len(s.config.BlockedWords) > 0 {
		lowerInput := strings.ToLower(input)
		for _, word := range s.config.BlockedWords {
			if strings.Contains(lowerInput, strings.ToLower(word)) {
				result.Allowed = false
				result.Checks = append(result.Checks, Check{
					Name:   "blocked_word",
					Passed: false,
					Reason: "Input contains blocked content",
				})
				return result, nil
			}
		}
		result.Checks = append(result.Checks, Check{Name: "blocked_word", Passed: true})
	}

	// Pattern check
	if len(s.config.BlockedPatterns) > 0 {
		for _, pattern := range s.config.BlockedPatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				continue // Skip invalid patterns
			}
			if re.MatchString(input) {
				result.Allowed = false
				result.Checks = append(result.Checks, Check{
					Name:   "pattern",
					Passed: false,
					Reason: "Input matches blocked pattern",
				})
				return result, nil
			}
		}
		result.Checks = append(result.Checks, Check{Name: "pattern", Passed: true})
	}

	// Moderation check (if enabled)
	if s.config.RequireModeration && s.moderator != nil {
		classification, err := s.moderator.Classify(ctx, input)
		if err != nil {
			if s.config.FailOpen {
				// Log error but allow request
				result.Checks = append(result.Checks, Check{
					Name:   "moderation",
					Passed: true,
					Reason: "Moderation service unavailable, allowing by default",
				})
			} else {
				result.Allowed = false
				result.Checks = append(result.Checks, Check{
					Name:   "moderation",
					Passed: false,
					Reason: "Moderation service unavailable",
				})
				return result, nil
			}
		} else {
			check := s.evaluateClassification(classification)
			result.Checks = append(result.Checks, check)
			if !check.Passed {
				result.Allowed = false
			}
		}
	}

	return result, nil
}

// CheckOutput validates agent output before sending to user.
func (s *Service) CheckOutput(ctx context.Context, output string) (*CheckResult, error) {
	if !s.config.Enabled {
		return &CheckResult{Allowed: true}, nil
	}

	result := &CheckResult{Allowed: true, Checks: []Check{}}

	// Length check (truncate rather than block)
	if len(output) > s.config.MaxOutputLength {
		result.Checks = append(result.Checks, Check{
			Name:   "length",
			Passed: true,
			Reason: fmt.Sprintf("Output truncated to %d characters", s.config.MaxOutputLength),
		})
		result.TruncatedOutput = output[:s.config.MaxOutputLength]
	} else {
		result.Checks = append(result.Checks, Check{Name: "length", Passed: true})
	}

	// Moderation check for output
	if s.config.RequireModeration && s.moderator != nil {
		classification, err := s.moderator.Classify(ctx, output)
		if err != nil {
			if s.config.FailOpen {
				result.Checks = append(result.Checks, Check{
					Name:   "moderation",
					Passed: true,
					Reason: "Moderation service unavailable",
				})
			} else {
				result.Allowed = false
				result.Checks = append(result.Checks, Check{
					Name:   "moderation",
					Passed: false,
					Reason: "Moderation service unavailable",
				})
			}
		} else {
			check := s.evaluateClassification(classification)
			result.Checks = append(result.Checks, check)
			if !check.Passed {
				result.Allowed = false
			}
		}
	}

	return result, nil
}

func (s *Service) evaluateClassification(c *Classification) Check {
	check := Check{Name: "moderation", Passed: true}
	thresholds := s.config.ModerationThresholds

	// Check each category against thresholds
	if c.Hate >= thresholds.Hate {
		check.Passed = false
		check.Reason = "Content flagged for hate speech"
		check.Category = "hate"
		check.Score = c.Hate
	} else if c.HateWithThreat >= thresholds.HateWithThreat {
		check.Passed = false
		check.Reason = "Content flagged for hate with threat"
		check.Category = "hate_threatening"
		check.Score = c.HateWithThreat
	} else if c.Violence >= thresholds.Violence {
		check.Passed = false
		check.Reason = "Content flagged for violence"
		check.Category = "violence"
		check.Score = c.Violence
	} else if c.ViolenceGraphic >= thresholds.ViolenceGraphic {
		check.Passed = false
		check.Reason = "Content flagged for graphic violence"
		check.Category = "violence_graphic"
		check.Score = c.ViolenceGraphic
	} else if c.Sexual >= thresholds.Sexual {
		check.Passed = false
		check.Reason = "Content flagged for sexual content"
		check.Category = "sexual"
		check.Score = c.Sexual
	} else if c.SelfHarm >= thresholds.SelfHarm {
		check.Passed = false
		check.Reason = "Content flagged for self-harm"
		check.Category = "self_harm"
		check.Score = c.SelfHarm
	}

	return check
}

// IsEnabled returns whether safety checking is enabled.
func (s *Service) IsEnabled() bool {
	return s.config.Enabled
}

// GetConfig returns the current configuration.
func (s *Service) GetConfig() Config {
	return s.config
}

// --- Types ---

// CheckResult represents the result of a safety check.
type CheckResult struct {
	Allowed         bool    `json:"allowed"`
	Checks          []Check `json:"checks"`
	TruncatedOutput string  `json:"truncated_output,omitempty"`
}

// Check represents a single safety check result.
type Check struct {
	Name     string  `json:"name"`
	Passed   bool    `json:"passed"`
	Reason   string  `json:"reason,omitempty"`
	Category string  `json:"category,omitempty"`
	Score    float64 `json:"score,omitempty"`
}

// Classification represents content moderation classification scores.
type Classification struct {
	Hate            float64 `json:"hate"`
	Violence        float64 `json:"violence"`
	Sexual          float64 `json:"sexual"`
	SelfHarm        float64 `json:"self_harm"`
	HateWithThreat  float64 `json:"hate_threatening"`
	ViolenceGraphic float64 `json:"violence_graphic"`
}

// SafetyError represents an error from the safety service.
type SafetyError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Check   *Check `json:"check,omitempty"`
}

func (e *SafetyError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewContentBlockedError creates a new content blocked error.
func NewContentBlockedError(check Check) *SafetyError {
	return &SafetyError{
		Code:    "content_blocked",
		Message: check.Reason,
		Check:   &check,
	}
}
