package tools

import "context"

type ToolInfo struct {
	Name        string
	Description string
	Parameters  map[string]any
	Required    []string
}

type toolResponseType string

const (
	ToolResponseTypeText  toolResponseType = "text"
	ToolResponseTypeImage toolResponseType = "image"
)

type ToolResponse struct {
	Type    toolResponseType `json:"type"`
	Content string           `json:"content"`
	IsError bool             `json:"is_error"`
}

func NewTextResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
	}
}

func NewTextErrorResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
		IsError: true,
	}
}

type BaseTool interface {
	Info() ToolInfo
	Run(ctx context.Context, params string) (ToolResponse, error)
}
