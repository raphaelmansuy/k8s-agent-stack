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

// Package ucm provides the Universal Content Model (UCM) for content translation.
// UCM abstracts content types across different LLM providers and agent protocols.
package ucm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
)

// ContentType represents the type of UCM content.
type ContentType string

const (
	// ContentTypeText is plain text content.
	ContentTypeText ContentType = "text"
	// ContentTypeImage is image content (base64 or URL).
	ContentTypeImage ContentType = "image"
	// ContentTypeAudio is audio content.
	ContentTypeAudio ContentType = "audio"
	// ContentTypeVideo is video content.
	ContentTypeVideo ContentType = "video"
	// ContentTypeDocument is document content (PDF, etc.).
	ContentTypeDocument ContentType = "document"
	// ContentTypeCode is source code content.
	ContentTypeCode ContentType = "code"
	// ContentTypeFunctionCall is a function/tool call.
	ContentTypeFunctionCall ContentType = "function_call"
	// ContentTypeFunctionResponse is a function/tool response.
	ContentTypeFunctionResponse ContentType = "function_response"
	// ContentTypeData is structured data (JSON, etc.).
	ContentTypeData ContentType = "data"
)

// Content represents a piece of UCM content.
type Content struct {
	Type     ContentType            `json:"type"`
	Text     string                 `json:"text,omitempty"`
	Data     interface{}            `json:"data,omitempty"`
	MimeType string                 `json:"mimeType,omitempty"`
	URL      string                 `json:"url,omitempty"`
	Encoding string                 `json:"encoding,omitempty"` // "base64", "url", etc.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewTextContent creates a new text content.
func NewTextContent(text string) *Content {
	return &Content{
		Type: ContentTypeText,
		Text: text,
	}
}

// NewCodeContent creates a new code content.
func NewCodeContent(code, language string) *Content {
	return &Content{
		Type:     ContentTypeCode,
		Text:     code,
		Metadata: map[string]interface{}{"language": language},
	}
}

// NewImageContent creates a new image content from base64 data.
func NewImageContent(data []byte, mimeType string) *Content {
	return &Content{
		Type:     ContentTypeImage,
		Data:     base64.StdEncoding.EncodeToString(data),
		MimeType: mimeType,
		Encoding: "base64",
	}
}

// NewImageURLContent creates a new image content from URL.
func NewImageURLContent(url, mimeType string) *Content {
	return &Content{
		Type:     ContentTypeImage,
		URL:      url,
		MimeType: mimeType,
		Encoding: "url",
	}
}

// NewFunctionCallContent creates a function call content.
func NewFunctionCallContent(id, name string, args map[string]interface{}) *Content {
	return &Content{
		Type: ContentTypeFunctionCall,
		Data: map[string]interface{}{
			"id":        id,
			"name":      name,
			"arguments": args,
		},
	}
}

// NewFunctionResponseContent creates a function response content.
func NewFunctionResponseContent(id, name string, result interface{}, err error) *Content {
	data := map[string]interface{}{
		"id":     id,
		"name":   name,
		"result": result,
	}
	if err != nil {
		data["error"] = err.Error()
	}
	return &Content{
		Type: ContentTypeFunctionResponse,
		Data: data,
	}
}

// NewDataContent creates a structured data content.
func NewDataContent(data interface{}) *Content {
	return &Content{
		Type: ContentTypeData,
		Data: data,
	}
}

// Message represents a UCM message.
type Message struct {
	ID        string                 `json:"id"`
	Role      Role                   `json:"role"`
	Contents  []*Content             `json:"contents"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Role represents the message sender role.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// NewMessage creates a new UCM message.
func NewMessage(id string, role Role, contents ...*Content) *Message {
	return &Message{
		ID:        id,
		Role:      role,
		Contents:  contents,
		Timestamp: time.Now(),
	}
}

// AddContent adds content to a message.
func (m *Message) AddContent(content *Content) {
	m.Contents = append(m.Contents, content)
}

// GetText returns the concatenated text content.
func (m *Message) GetText() string {
	var texts []string
	for _, c := range m.Contents {
		if c.Type == ContentTypeText || c.Type == ContentTypeCode {
			texts = append(texts, c.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// === Translator Interface ===

// Translator translates between UCM and provider-specific formats.
type Translator interface {
	// ToUCM converts provider-specific content to UCM.
	ToUCM(input interface{}) (*Message, error)
	// FromUCM converts UCM message to provider-specific format.
	FromUCM(msg *Message) (interface{}, error)
}

// === A2A Translator ===

// A2ATranslator translates between UCM and A2A protocol.
type A2ATranslator struct{}

// NewA2ATranslator creates a new A2A translator.
func NewA2ATranslator() *A2ATranslator {
	return &A2ATranslator{}
}

// ToUCM converts A2A message to UCM format.
func (t *A2ATranslator) ToUCM(input interface{}) (*Message, error) {
	msg, ok := input.(*a2a.Message)
	if !ok {
		return nil, fmt.Errorf("expected *a2a.Message, got %T", input)
	}

	role := RoleAssistant
	if msg.Role == "user" {
		role = RoleUser
	}

	ucmMsg := NewMessage(msg.MessageID, role)
	ucmMsg.Metadata = msg.Metadata

	for _, part := range msg.Parts {
		content := t.partToContent(part)
		if content != nil {
			ucmMsg.AddContent(content)
		}
	}

	return ucmMsg, nil
}

// partToContent converts an A2A part to UCM content.
func (t *A2ATranslator) partToContent(part a2a.Part) *Content {
	switch part.Kind {
	case "text":
		return NewTextContent(part.Text)
	case "data":
		// Check for function call/response
		if part.Metadata != nil {
			if kagentType, ok := part.Metadata["kagent_type"].(string); ok {
				switch kagentType {
				case "function_call":
					id, _ := part.Data["id"].(string)
					name, _ := part.Data["name"].(string)
					args, _ := part.Data["args"].(map[string]interface{})
					return NewFunctionCallContent(id, name, args)
				case "function_response":
					id, _ := part.Data["id"].(string)
					name, _ := part.Data["name"].(string)
					response := part.Data["response"]
					return NewFunctionResponseContent(id, name, response, nil)
				}
			}
		}
		return NewDataContent(part.Data)
	default:
		return nil
	}
}

// FromUCM converts UCM message to A2A format.
func (t *A2ATranslator) FromUCM(msg *Message) (interface{}, error) {
	role := "agent"
	if msg.Role == RoleUser {
		role = "user"
	}

	parts := make([]a2a.Part, 0, len(msg.Contents))
	for _, content := range msg.Contents {
		part := t.contentToPart(content)
		parts = append(parts, part)
	}

	return a2a.NewMessage(msg.ID, role, parts), nil
}

// contentToPart converts UCM content to A2A part.
func (t *A2ATranslator) contentToPart(content *Content) a2a.Part {
	switch content.Type {
	case ContentTypeText, ContentTypeCode:
		return a2a.TextPart(content.Text)
	case ContentTypeFunctionCall:
		data, _ := content.Data.(map[string]interface{})
		id, _ := data["id"].(string)
		name, _ := data["name"].(string)
		args, _ := data["arguments"].(map[string]interface{})
		return a2a.FunctionCallPart(id, name, args)
	case ContentTypeFunctionResponse:
		data, _ := content.Data.(map[string]interface{})
		id, _ := data["id"].(string)
		name, _ := data["name"].(string)
		result := data["result"]
		return a2a.FunctionResponsePart(id, name, result)
	case ContentTypeData:
		dataMap, ok := content.Data.(map[string]interface{})
		if !ok {
			// Convert to map via JSON
			jsonData, _ := json.Marshal(content.Data)
			_ = json.Unmarshal(jsonData, &dataMap)
		}
		return a2a.DataPart(dataMap)
	default:
		// Default to data part
		return a2a.DataPart(map[string]interface{}{
			"type": string(content.Type),
			"data": content.Data,
			"url":  content.URL,
		})
	}
}

// === OpenAI Translator ===

// OpenAITranslator translates between UCM and OpenAI format.
type OpenAITranslator struct{}

// OpenAIMessage represents an OpenAI chat message.
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"` // string or []OpenAIContentPart
	Name       string           `json:"name,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

// OpenAIContentPart represents a content part in OpenAI format.
type OpenAIContentPart struct {
	Type     string          `json:"type"` // "text", "image_url"
	Text     string          `json:"text,omitempty"`
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"`
}

// OpenAIImageURL represents an image URL in OpenAI format.
type OpenAIImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// OpenAIToolCall represents a tool call in OpenAI format.
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIFunctionCall represents a function call in OpenAI format.
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// NewOpenAITranslator creates a new OpenAI translator.
func NewOpenAITranslator() *OpenAITranslator {
	return &OpenAITranslator{}
}

// ToUCM converts OpenAI message to UCM format.
func (t *OpenAITranslator) ToUCM(input interface{}) (*Message, error) {
	msg, ok := input.(*OpenAIMessage)
	if !ok {
		return nil, fmt.Errorf("expected *OpenAIMessage, got %T", input)
	}

	role := t.openaiRoleToUCM(msg.Role)
	ucmMsg := NewMessage("", role)

	// Handle content
	switch c := msg.Content.(type) {
	case string:
		ucmMsg.AddContent(NewTextContent(c))
	case []interface{}:
		for _, part := range c {
			if partMap, ok := part.(map[string]interface{}); ok {
				if content := t.openaiPartToContent(partMap); content != nil {
					ucmMsg.AddContent(content)
				}
			}
		}
	}

	// Handle tool calls
	for _, tc := range msg.ToolCalls {
		var args map[string]interface{}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		ucmMsg.AddContent(NewFunctionCallContent(tc.ID, tc.Function.Name, args))
	}

	return ucmMsg, nil
}

// openaiRoleToUCM converts OpenAI role to UCM role.
func (t *OpenAITranslator) openaiRoleToUCM(role string) Role {
	switch role {
	case "user":
		return RoleUser
	case "assistant":
		return RoleAssistant
	case "system":
		return RoleSystem
	case "tool":
		return RoleTool
	default:
		return RoleUser
	}
}

// openaiPartToContent converts OpenAI content part to UCM content.
func (t *OpenAITranslator) openaiPartToContent(part map[string]interface{}) *Content {
	partType, _ := part["type"].(string)
	switch partType {
	case "text":
		text, _ := part["text"].(string)
		return NewTextContent(text)
	case "image_url":
		if imgURL, ok := part["image_url"].(map[string]interface{}); ok {
			url, _ := imgURL["url"].(string)
			return NewImageURLContent(url, "image/jpeg")
		}
	}
	return nil
}

// FromUCM converts UCM message to OpenAI format.
func (t *OpenAITranslator) FromUCM(msg *Message) (interface{}, error) {
	role := t.ucmRoleToOpenAI(msg.Role)

	openaiMsg := &OpenAIMessage{
		Role: role,
	}

	// Collect content and tool calls
	var textParts []OpenAIContentPart
	var toolCalls []OpenAIToolCall

	for _, content := range msg.Contents {
		switch content.Type {
		case ContentTypeText:
			textParts = append(textParts, OpenAIContentPart{
				Type: "text",
				Text: content.Text,
			})
		case ContentTypeCode:
			textParts = append(textParts, OpenAIContentPart{
				Type: "text",
				Text: fmt.Sprintf("```%s\n%s\n```",
					content.Metadata["language"], content.Text),
			})
		case ContentTypeImage:
			var url string
			if content.Encoding == "base64" {
				url = fmt.Sprintf("data:%s;base64,%s", content.MimeType, content.Data)
			} else {
				url = content.URL
			}
			textParts = append(textParts, OpenAIContentPart{
				Type:     "image_url",
				ImageURL: &OpenAIImageURL{URL: url},
			})
		case ContentTypeFunctionCall:
			data, _ := content.Data.(map[string]interface{})
			id, _ := data["id"].(string)
			name, _ := data["name"].(string)
			args, _ := data["arguments"].(map[string]interface{})
			argsJSON, _ := json.Marshal(args)
			toolCalls = append(toolCalls, OpenAIToolCall{
				ID:   id,
				Type: "function",
				Function: OpenAIFunctionCall{
					Name:      name,
					Arguments: string(argsJSON),
				},
			})
		case ContentTypeFunctionResponse:
			data, _ := content.Data.(map[string]interface{})
			id, _ := data["id"].(string)
			result := data["result"]
			resultJSON, _ := json.Marshal(result)
			// Return as tool message
			return &OpenAIMessage{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: id,
			}, nil
		default:
			// Skip other content types
		}
	}

	// Set content
	if len(textParts) == 1 && textParts[0].Type == "text" {
		openaiMsg.Content = textParts[0].Text
	} else if len(textParts) > 0 {
		openaiMsg.Content = textParts
	}

	// Set tool calls
	if len(toolCalls) > 0 {
		openaiMsg.ToolCalls = toolCalls
	}

	return openaiMsg, nil
}

// ucmRoleToOpenAI converts UCM role to OpenAI role.
func (t *OpenAITranslator) ucmRoleToOpenAI(role Role) string {
	switch role {
	case RoleUser:
		return "user"
	case RoleAssistant:
		return "assistant"
	case RoleSystem:
		return "system"
	case RoleTool:
		return "tool"
	default:
		return "user"
	}
}

// === Anthropic Translator ===

// AnthropicTranslator translates between UCM and Anthropic Claude format.
type AnthropicTranslator struct{}

// AnthropicMessage represents an Anthropic message.
type AnthropicMessage struct {
	Role    string             `json:"role"`
	Content []AnthropicContent `json:"content"`
}

// AnthropicContent represents content in Anthropic format.
type AnthropicContent struct {
	Type      string           `json:"type"` // "text", "image", "tool_use", "tool_result"
	Text      string           `json:"text,omitempty"`
	Source    *AnthropicSource `json:"source,omitempty"`
	ID        string           `json:"id,omitempty"`
	Name      string           `json:"name,omitempty"`
	Input     interface{}      `json:"input,omitempty"`
	ToolUseID string           `json:"tool_use_id,omitempty"`
	Content   string           `json:"content,omitempty"`
}

// AnthropicSource represents image source in Anthropic format.
type AnthropicSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// NewAnthropicTranslator creates a new Anthropic translator.
func NewAnthropicTranslator() *AnthropicTranslator {
	return &AnthropicTranslator{}
}

// ToUCM converts Anthropic message to UCM format.
func (t *AnthropicTranslator) ToUCM(input interface{}) (*Message, error) {
	msg, ok := input.(*AnthropicMessage)
	if !ok {
		return nil, fmt.Errorf("expected *AnthropicMessage, got %T", input)
	}

	role := RoleUser
	if msg.Role == "assistant" {
		role = RoleAssistant
	}

	ucmMsg := NewMessage("", role)

	for _, content := range msg.Content {
		switch content.Type {
		case "text":
			ucmMsg.AddContent(NewTextContent(content.Text))
		case "image":
			if content.Source != nil {
				data, _ := base64.StdEncoding.DecodeString(content.Source.Data)
				ucmMsg.AddContent(NewImageContent(data, content.Source.MediaType))
			}
		case "tool_use":
			args, _ := content.Input.(map[string]interface{})
			ucmMsg.AddContent(NewFunctionCallContent(content.ID, content.Name, args))
		case "tool_result":
			ucmMsg.AddContent(NewFunctionResponseContent(
				content.ToolUseID, "", content.Content, nil))
		}
	}

	return ucmMsg, nil
}

// FromUCM converts UCM message to Anthropic format.
func (t *AnthropicTranslator) FromUCM(msg *Message) (interface{}, error) {
	role := "user"
	if msg.Role == RoleAssistant {
		role = "assistant"
	}

	var contents []AnthropicContent

	for _, content := range msg.Contents {
		switch content.Type {
		case ContentTypeText:
			contents = append(contents, AnthropicContent{
				Type: "text",
				Text: content.Text,
			})
		case ContentTypeCode:
			contents = append(contents, AnthropicContent{
				Type: "text",
				Text: fmt.Sprintf("```%s\n%s\n```",
					content.Metadata["language"], content.Text),
			})
		case ContentTypeImage:
			if content.Encoding == "base64" {
				dataStr, _ := content.Data.(string)
				contents = append(contents, AnthropicContent{
					Type: "image",
					Source: &AnthropicSource{
						Type:      "base64",
						MediaType: content.MimeType,
						Data:      dataStr,
					},
				})
			}
		case ContentTypeFunctionCall:
			data, _ := content.Data.(map[string]interface{})
			id, _ := data["id"].(string)
			name, _ := data["name"].(string)
			contents = append(contents, AnthropicContent{
				Type:  "tool_use",
				ID:    id,
				Name:  name,
				Input: data["arguments"],
			})
		case ContentTypeFunctionResponse:
			data, _ := content.Data.(map[string]interface{})
			resultJSON, _ := json.Marshal(data["result"])
			id, _ := data["id"].(string)
			contents = append(contents, AnthropicContent{
				Type:      "tool_result",
				ToolUseID: id,
				Content:   string(resultJSON),
			})
		default:
			// Skip other content types
		}
	}

	return &AnthropicMessage{
		Role:    role,
		Content: contents,
	}, nil
}
