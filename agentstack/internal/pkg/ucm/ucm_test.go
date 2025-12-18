package ucm

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
)

func TestNewTextContent(t *testing.T) {
	content := NewTextContent("Hello, world!")

	if content.Type != ContentTypeText {
		t.Errorf("expected type 'text', got '%s'", content.Type)
	}
	if content.Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got '%s'", content.Text)
	}
}

func TestNewCodeContent(t *testing.T) {
	content := NewCodeContent("fmt.Println(\"Hello\")", "go")

	if content.Type != ContentTypeCode {
		t.Errorf("expected type 'code', got '%s'", content.Type)
	}
	if content.Metadata["language"] != "go" {
		t.Errorf("expected language 'go', got '%v'", content.Metadata["language"])
	}
}

func TestNewImageContent(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF} // JPEG magic bytes
	content := NewImageContent(data, "image/jpeg")

	if content.Type != ContentTypeImage {
		t.Errorf("expected type 'image', got '%s'", content.Type)
	}
	if content.MimeType != "image/jpeg" {
		t.Errorf("expected mimeType 'image/jpeg', got '%s'", content.MimeType)
	}
	if content.Encoding != "base64" {
		t.Errorf("expected encoding 'base64', got '%s'", content.Encoding)
	}

	// Verify base64 encoding
	decoded, err := base64.StdEncoding.DecodeString(content.Data.(string))
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	if len(decoded) != 3 {
		t.Errorf("expected 3 bytes, got %d", len(decoded))
	}
}

func TestNewImageURLContent(t *testing.T) {
	content := NewImageURLContent("https://example.com/image.jpg", "image/jpeg")

	if content.Type != ContentTypeImage {
		t.Errorf("expected type 'image', got '%s'", content.Type)
	}
	if content.URL != "https://example.com/image.jpg" {
		t.Errorf("expected URL, got '%s'", content.URL)
	}
	if content.Encoding != "url" {
		t.Errorf("expected encoding 'url', got '%s'", content.Encoding)
	}
}

func TestNewFunctionCallContent(t *testing.T) {
	args := map[string]interface{}{"city": "Paris"}
	content := NewFunctionCallContent("call-123", "get_weather", args)

	if content.Type != ContentTypeFunctionCall {
		t.Errorf("expected type 'function_call', got '%s'", content.Type)
	}

	data, ok := content.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be map")
	}
	if data["name"] != "get_weather" {
		t.Errorf("expected name 'get_weather', got '%v'", data["name"])
	}
}

func TestNewFunctionResponseContent(t *testing.T) {
	result := map[string]interface{}{"temperature": 20}
	content := NewFunctionResponseContent("call-123", "get_weather", result, nil)

	if content.Type != ContentTypeFunctionResponse {
		t.Errorf("expected type 'function_response', got '%s'", content.Type)
	}

	data, ok := content.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be map")
	}
	if _, hasError := data["error"]; hasError {
		t.Error("expected no error field")
	}
}

func TestNewMessage(t *testing.T) {
	content := NewTextContent("Hello!")
	msg := NewMessage("msg-123", RoleUser, content)

	if msg.ID != "msg-123" {
		t.Errorf("expected id 'msg-123', got '%s'", msg.ID)
	}
	if msg.Role != RoleUser {
		t.Errorf("expected role 'user', got '%s'", msg.Role)
	}
	if len(msg.Contents) != 1 {
		t.Errorf("expected 1 content, got %d", len(msg.Contents))
	}
}

func TestMessageAddContent(t *testing.T) {
	msg := NewMessage("msg-123", RoleUser)
	msg.AddContent(NewTextContent("Part 1"))
	msg.AddContent(NewTextContent("Part 2"))

	if len(msg.Contents) != 2 {
		t.Errorf("expected 2 contents, got %d", len(msg.Contents))
	}
}

func TestMessageGetText(t *testing.T) {
	msg := NewMessage("msg-123", RoleUser,
		NewTextContent("Hello"),
		NewCodeContent("print('world')", "python"),
		NewDataContent(map[string]interface{}{"ignore": true}),
	)

	text := msg.GetText()
	if text != "Hello\nprint('world')" {
		t.Errorf("expected 'Hello\\nprint('world')', got '%s'", text)
	}
}

func TestA2ATranslatorToUCM(t *testing.T) {
	translator := NewA2ATranslator()

	a2aMsg := a2a.NewMessage("msg-123", "user", []a2a.Part{
		a2a.TextPart("Hello, agent!"),
		a2a.DataPart(map[string]interface{}{"key": "value"}),
	})

	ucmMsg, err := translator.ToUCM(a2aMsg)
	if err != nil {
		t.Fatalf("failed to translate: %v", err)
	}

	if ucmMsg.Role != RoleUser {
		t.Errorf("expected role 'user', got '%s'", ucmMsg.Role)
	}
	if len(ucmMsg.Contents) != 2 {
		t.Errorf("expected 2 contents, got %d", len(ucmMsg.Contents))
	}
	if ucmMsg.Contents[0].Type != ContentTypeText {
		t.Errorf("expected first content type 'text', got '%s'", ucmMsg.Contents[0].Type)
	}
}

func TestA2ATranslatorFromUCM(t *testing.T) {
	translator := NewA2ATranslator()

	ucmMsg := NewMessage("msg-123", RoleAssistant,
		NewTextContent("Hello, user!"),
		NewFunctionCallContent("call-1", "get_weather", map[string]interface{}{"city": "Paris"}),
	)

	result, err := translator.FromUCM(ucmMsg)
	if err != nil {
		t.Fatalf("failed to translate: %v", err)
	}

	a2aMsg, ok := result.(*a2a.Message)
	if !ok {
		t.Fatalf("expected *a2a.Message, got %T", result)
	}

	if a2aMsg.Role != "agent" {
		t.Errorf("expected role 'agent', got '%s'", a2aMsg.Role)
	}
	if len(a2aMsg.Parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(a2aMsg.Parts))
	}
}

func TestA2ATranslatorFunctionCall(t *testing.T) {
	translator := NewA2ATranslator()

	// Create A2A function call part
	a2aMsg := &a2a.Message{
		Kind:      "message",
		MessageID: "msg-123",
		Role:      "agent",
		Parts: []a2a.Part{
			a2a.FunctionCallPart("call-1", "get_weather", map[string]interface{}{"city": "London"}),
		},
	}

	ucmMsg, err := translator.ToUCM(a2aMsg)
	if err != nil {
		t.Fatalf("failed to translate: %v", err)
	}

	if len(ucmMsg.Contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(ucmMsg.Contents))
	}
	if ucmMsg.Contents[0].Type != ContentTypeFunctionCall {
		t.Errorf("expected type 'function_call', got '%s'", ucmMsg.Contents[0].Type)
	}
}

func TestOpenAITranslatorToUCM(t *testing.T) {
	translator := NewOpenAITranslator()

	openaiMsg := &OpenAIMessage{
		Role:    "user",
		Content: "What's the weather in Paris?",
	}

	ucmMsg, err := translator.ToUCM(openaiMsg)
	if err != nil {
		t.Fatalf("failed to translate: %v", err)
	}

	if ucmMsg.Role != RoleUser {
		t.Errorf("expected role 'user', got '%s'", ucmMsg.Role)
	}
	if len(ucmMsg.Contents) != 1 {
		t.Errorf("expected 1 content, got %d", len(ucmMsg.Contents))
	}
	if ucmMsg.Contents[0].Text != "What's the weather in Paris?" {
		t.Errorf("expected text, got '%s'", ucmMsg.Contents[0].Text)
	}
}

func TestOpenAITranslatorFromUCM(t *testing.T) {
	translator := NewOpenAITranslator()

	ucmMsg := NewMessage("msg-123", RoleAssistant,
		NewTextContent("The weather in Paris is sunny."),
	)

	result, err := translator.FromUCM(ucmMsg)
	if err != nil {
		t.Fatalf("failed to translate: %v", err)
	}

	openaiMsg, ok := result.(*OpenAIMessage)
	if !ok {
		t.Fatalf("expected *OpenAIMessage, got %T", result)
	}

	if openaiMsg.Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", openaiMsg.Role)
	}

	content, ok := openaiMsg.Content.(string)
	if !ok {
		t.Fatalf("expected string content, got %T", openaiMsg.Content)
	}
	if content != "The weather in Paris is sunny." {
		t.Errorf("expected text, got '%s'", content)
	}
}

func TestOpenAITranslatorToolCalls(t *testing.T) {
	translator := NewOpenAITranslator()

	openaiMsg := &OpenAIMessage{
		Role:    "assistant",
		Content: nil,
		ToolCalls: []OpenAIToolCall{
			{
				ID:   "call-123",
				Type: "function",
				Function: OpenAIFunctionCall{
					Name:      "get_weather",
					Arguments: `{"city":"Paris"}`,
				},
			},
		},
	}

	ucmMsg, err := translator.ToUCM(openaiMsg)
	if err != nil {
		t.Fatalf("failed to translate: %v", err)
	}

	if len(ucmMsg.Contents) != 1 {
		t.Errorf("expected 1 content, got %d", len(ucmMsg.Contents))
	}
	if ucmMsg.Contents[0].Type != ContentTypeFunctionCall {
		t.Errorf("expected type 'function_call', got '%s'", ucmMsg.Contents[0].Type)
	}
}

func TestAnthropicTranslatorToUCM(t *testing.T) {
	translator := NewAnthropicTranslator()

	anthropicMsg := &AnthropicMessage{
		Role: "user",
		Content: []AnthropicContent{
			{Type: "text", Text: "What's the weather?"},
		},
	}

	ucmMsg, err := translator.ToUCM(anthropicMsg)
	if err != nil {
		t.Fatalf("failed to translate: %v", err)
	}

	if ucmMsg.Role != RoleUser {
		t.Errorf("expected role 'user', got '%s'", ucmMsg.Role)
	}
	if len(ucmMsg.Contents) != 1 {
		t.Errorf("expected 1 content, got %d", len(ucmMsg.Contents))
	}
}

func TestAnthropicTranslatorFromUCM(t *testing.T) {
	translator := NewAnthropicTranslator()

	ucmMsg := NewMessage("msg-123", RoleAssistant,
		NewTextContent("The weather is sunny."),
	)

	result, err := translator.FromUCM(ucmMsg)
	if err != nil {
		t.Fatalf("failed to translate: %v", err)
	}

	anthropicMsg, ok := result.(*AnthropicMessage)
	if !ok {
		t.Fatalf("expected *AnthropicMessage, got %T", result)
	}

	if anthropicMsg.Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", anthropicMsg.Role)
	}
	if len(anthropicMsg.Content) != 1 {
		t.Errorf("expected 1 content, got %d", len(anthropicMsg.Content))
	}
}

func TestContentTypesSerialization(t *testing.T) {
	types := []ContentType{
		ContentTypeText,
		ContentTypeImage,
		ContentTypeAudio,
		ContentTypeVideo,
		ContentTypeDocument,
		ContentTypeCode,
		ContentTypeFunctionCall,
		ContentTypeFunctionResponse,
		ContentTypeData,
	}

	for _, ct := range types {
		content := &Content{Type: ct, Text: "test"}
		data, err := json.Marshal(content)
		if err != nil {
			t.Errorf("failed to marshal content type %s: %v", ct, err)
		}

		var parsed Content
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Errorf("failed to unmarshal content type %s: %v", ct, err)
		}
		if parsed.Type != ct {
			t.Errorf("expected type %s, got %s", ct, parsed.Type)
		}
	}
}

func TestRolesSerialization(t *testing.T) {
	roles := []Role{
		RoleUser,
		RoleAssistant,
		RoleSystem,
		RoleTool,
	}

	for _, role := range roles {
		msg := NewMessage("test", role, NewTextContent("test"))
		data, err := json.Marshal(msg)
		if err != nil {
			t.Errorf("failed to marshal role %s: %v", role, err)
		}

		var parsed Message
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Errorf("failed to unmarshal role %s: %v", role, err)
		}
		if parsed.Role != role {
			t.Errorf("expected role %s, got %s", role, parsed.Role)
		}
	}
}
