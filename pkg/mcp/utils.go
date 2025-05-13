package mcp

import (
	"encoding/json"
	"fmt"
)

func ParseCallToolResult(rawMessage *json.RawMessage) (*CallToolResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var result CallToolResult

	isError, ok := jsonContent["isError"]
	if ok {
		if isErrorBool, ok := isError.(bool); ok {
			result.IsError = isErrorBool
		}
	}

	contents, ok := jsonContent["content"]
	if !ok {
		return nil, fmt.Errorf("content is missing")
	}

	contentArr, ok := contents.([]any)
	if !ok {
		return nil, fmt.Errorf("content is not an array")
	}

	for _, content := range contentArr {
		// Extract content
		contentMap, ok := content.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content is not an object")
		}

		// Process content
		content, err := ParseContent(contentMap)
		if err != nil {
			return nil, err
		}

		result.Content = append(result.Content, content)
	}

	return &result, nil
}

func ParseContent(contentMap map[string]any) (Content, error) {
	contentType := ExtractString(contentMap, "type")

	switch contentType {
	case "text":
		text := ExtractString(contentMap, "text")
		return &TextContent{
			Type: "text",
			Text: text,
		}, nil

	case "image":
		data := ExtractString(contentMap, "data")
		mimeType := ExtractString(contentMap, "mimeType")
		if data == "" || mimeType == "" {
			return nil, fmt.Errorf("image data or mimeType is missing")
		}
		return &ImageContent{
			Type:     "image",
			Data:     data,
			MimeType: mimeType,
		}, nil
	}

	return nil, fmt.Errorf("unsupported content type: %s", contentType)
}

func ExtractString(data map[string]any, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func CoverToStdioClientEnv(env map[string]string) []string {
	envList := []string{}
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	return envList
}

// NewTextContent
// Helper function to create a new TextContent
func NewTextContent(text string) *TextContent {
	return &TextContent{
		Type: "text",
		Text: text,
	}
}

// NewImageContent
// Helper function to create a new ImageContent
func NewImageContent(data, mimeType string) *ImageContent {
	return &ImageContent{
		Type:     "image",
		Data:     data,
		MimeType: mimeType,
	}
}
