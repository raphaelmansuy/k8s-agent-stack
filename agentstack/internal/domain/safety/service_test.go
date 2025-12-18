package safety

import (
	"context"
	"errors"
	"testing"
)

// Mock moderator for testing
type mockModerator struct {
	classification *Classification
	err            error
}

func (m *mockModerator) Classify(ctx context.Context, text string) (*Classification, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.classification != nil {
		return m.classification, nil
	}
	return &Classification{}, nil
}

func TestNewService(t *testing.T) {
	config := DefaultConfig()
	svc := NewService(config, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestCheckInput_Disabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false
	svc := NewService(config, nil)

	result, err := svc.CheckInput(context.Background(), "any input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed when safety is disabled")
	}
}

func TestCheckInput_MaxLength(t *testing.T) {
	config := DefaultConfig()
	config.MaxInputLength = 10
	svc := NewService(config, nil)

	// Input within limit
	result, err := svc.CheckInput(context.Background(), "short")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed for short input")
	}

	// Input exceeds limit
	result, err = svc.CheckInput(context.Background(), "this is a very long input string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected blocked for long input")
	}
	if len(result.Checks) == 0 || result.Checks[0].Name != "length" {
		t.Error("expected length check to fail")
	}
}

func TestCheckInput_BlockedWords(t *testing.T) {
	config := DefaultConfig()
	config.BlockedWords = []string{"badword", "forbidden"}
	svc := NewService(config, nil)

	// Clean input
	result, err := svc.CheckInput(context.Background(), "This is a clean message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed for clean input")
	}

	// Input with blocked word
	result, err = svc.CheckInput(context.Background(), "This contains badword in it")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected blocked for input with blocked word")
	}

	// Case insensitive
	result, err = svc.CheckInput(context.Background(), "This contains BADWORD in it")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected blocked for case-insensitive match")
	}
}

func TestCheckInput_BlockedPatterns(t *testing.T) {
	config := DefaultConfig()
	config.BlockedPatterns = []string{`\b[A-Z]{5,}\b`} // Blocks words with 5+ uppercase letters
	svc := NewService(config, nil)

	// Clean input
	result, err := svc.CheckInput(context.Background(), "This is a normal message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed for clean input")
	}

	// Input matching pattern
	result, err = svc.CheckInput(context.Background(), "This has BLOCKED pattern")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected blocked for pattern match")
	}
}

func TestCheckInput_Moderation(t *testing.T) {
	config := DefaultConfig()
	config.RequireModeration = true

	// Test with safe content
	moderator := &mockModerator{
		classification: &Classification{
			Hate:     0.1,
			Violence: 0.1,
		},
	}
	svc := NewService(config, moderator)

	result, err := svc.CheckInput(context.Background(), "Hello, how are you?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed for safe content")
	}

	// Test with hate content
	moderator.classification = &Classification{
		Hate: 0.9, // Above threshold
	}

	result, err = svc.CheckInput(context.Background(), "Hateful content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected blocked for hate content")
	}
}

func TestCheckInput_ModerationFailOpen(t *testing.T) {
	config := DefaultConfig()
	config.RequireModeration = true
	config.FailOpen = true

	moderator := &mockModerator{
		err: errors.New("moderation service unavailable"),
	}
	svc := NewService(config, moderator)

	result, err := svc.CheckInput(context.Background(), "Some input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed when fail open is enabled")
	}
}

func TestCheckInput_ModerationFailClosed(t *testing.T) {
	config := DefaultConfig()
	config.RequireModeration = true
	config.FailOpen = false

	moderator := &mockModerator{
		err: errors.New("moderation service unavailable"),
	}
	svc := NewService(config, moderator)

	result, err := svc.CheckInput(context.Background(), "Some input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected blocked when fail open is disabled")
	}
}

func TestCheckOutput_Disabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false
	svc := NewService(config, nil)

	result, err := svc.CheckOutput(context.Background(), "any output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed when safety is disabled")
	}
}

func TestCheckOutput_MaxLength(t *testing.T) {
	config := DefaultConfig()
	config.MaxOutputLength = 20
	svc := NewService(config, nil)

	// Output exceeding limit should be truncated, not blocked
	result, err := svc.CheckOutput(context.Background(), "this is a very long output string that exceeds the limit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed for long output (with truncation)")
	}
	if result.TruncatedOutput == "" {
		t.Error("expected truncated output")
	}
	if len(result.TruncatedOutput) != 20 {
		t.Errorf("expected truncated output length 20, got %d", len(result.TruncatedOutput))
	}
}

func TestCheckOutput_Moderation(t *testing.T) {
	config := DefaultConfig()
	config.RequireModeration = true

	// Test with violent content
	moderator := &mockModerator{
		classification: &Classification{
			Violence: 0.9, // Above threshold
		},
	}
	svc := NewService(config, moderator)

	result, err := svc.CheckOutput(context.Background(), "Violent content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected blocked for violent content")
	}
}

func TestEvaluateClassification(t *testing.T) {
	config := DefaultConfig()
	svc := NewService(config, nil)

	tests := []struct {
		name           string
		classification *Classification
		expectPassed   bool
		expectCategory string
	}{
		{
			name:           "safe content",
			classification: &Classification{Hate: 0.1, Violence: 0.1},
			expectPassed:   true,
		},
		{
			name:           "hate content",
			classification: &Classification{Hate: 0.9},
			expectPassed:   false,
			expectCategory: "hate",
		},
		{
			name:           "violence content",
			classification: &Classification{Violence: 0.9},
			expectPassed:   false,
			expectCategory: "violence",
		},
		{
			name:           "sexual content",
			classification: &Classification{Sexual: 0.9},
			expectPassed:   false,
			expectCategory: "sexual",
		},
		{
			name:           "self-harm content",
			classification: &Classification{SelfHarm: 0.9},
			expectPassed:   false,
			expectCategory: "self_harm",
		},
		{
			name:           "hate with threat",
			classification: &Classification{HateWithThreat: 0.7},
			expectPassed:   false,
			expectCategory: "hate_threatening",
		},
		{
			name:           "graphic violence",
			classification: &Classification{ViolenceGraphic: 0.7},
			expectPassed:   false,
			expectCategory: "violence_graphic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := svc.evaluateClassification(tt.classification)
			if check.Passed != tt.expectPassed {
				t.Errorf("expected Passed=%v, got %v", tt.expectPassed, check.Passed)
			}
			if !tt.expectPassed && check.Category != tt.expectCategory {
				t.Errorf("expected category '%s', got '%s'", tt.expectCategory, check.Category)
			}
		})
	}
}

func TestIsEnabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	svc := NewService(config, nil)
	if !svc.IsEnabled() {
		t.Error("expected IsEnabled to return true")
	}

	config.Enabled = false
	svc = NewService(config, nil)
	if svc.IsEnabled() {
		t.Error("expected IsEnabled to return false")
	}
}

func TestGetConfig(t *testing.T) {
	config := DefaultConfig()
	config.MaxInputLength = 12345
	svc := NewService(config, nil)

	retrieved := svc.GetConfig()
	if retrieved.MaxInputLength != 12345 {
		t.Errorf("expected MaxInputLength 12345, got %d", retrieved.MaxInputLength)
	}
}

func TestSafetyError(t *testing.T) {
	err := &SafetyError{
		Code:    "test_error",
		Message: "Test error message",
	}

	expected := "test_error: Test error message"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}
}

func TestNewContentBlockedError(t *testing.T) {
	check := Check{
		Name:     "moderation",
		Passed:   false,
		Reason:   "Content flagged for hate",
		Category: "hate",
	}

	err := NewContentBlockedError(check)
	if err.Code != "content_blocked" {
		t.Errorf("expected code 'content_blocked', got '%s'", err.Code)
	}
	if err.Check.Category != "hate" {
		t.Errorf("expected category 'hate', got '%s'", err.Check.Category)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("expected Enabled=true by default")
	}
	if !config.FailOpen {
		t.Error("expected FailOpen=true by default")
	}
	if config.MaxInputLength != 32000 {
		t.Errorf("expected MaxInputLength=32000, got %d", config.MaxInputLength)
	}
	if config.ModerationThresholds.Hate != 0.8 {
		t.Errorf("expected Hate threshold 0.8, got %f", config.ModerationThresholds.Hate)
	}
}
