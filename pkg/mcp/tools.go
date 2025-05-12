package mcp

// NewTextContent TextContentType is the content type for text data
func NewTextContent(textData string) *TextContent {
	return &TextContent{
		Type: TextContentType,
		Text: textData,
	}
}

// NewImageContent ImageContentType is the content type for image data
func NewImageContent(imageData, mimeType string) *ImageContent {
	return &ImageContent{
		Type:     ImageContentType,
		Data:     imageData,
		MimeType: mimeType,
	}
}

// NewAudioContent AudioContentType is the content type for audio data
func NewAudioContent(audioData string, mimeType string) *AudioContent {
	return &AudioContent{
		Type:     AudioContentType,
		Data:     audioData,
		MimeType: mimeType,
	}
}

func NewCallToolResult(content []Content, isError bool) *CallToolResult {
	return &CallToolResult{
		Content: content,
		IsError: isError,
	}
}
