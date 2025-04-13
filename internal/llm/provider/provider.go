package provider

import (
	"context"

	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/message"
)

// EventType represents the type of streaming event
type EventType string

const (
	EventContentStart  EventType = "content_start"
	EventContentDelta  EventType = "content_delta"
	EventThinkingDelta EventType = "thinking_delta"
	EventContentStop   EventType = "content_stop"
	EventComplete      EventType = "complete"
	EventError         EventType = "error"
	EventWarning       EventType = "warning"
	EventInfo          EventType = "info"
)

type TokenUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

type ProviderResponse struct {
	Content      string
	ToolCalls    []message.ToolCall
	Usage        TokenUsage
	FinishReason string
}

type ProviderEvent struct {
	Type     EventType
	Content  string
	Thinking string
	ToolCall *message.ToolCall
	Error    error
	Response *ProviderResponse

	// Used for giving users info on e.x retry
	Info string
}

type Provider interface {
	SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error)

	StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (<-chan ProviderEvent, error)
}

func cleanupMessages(messages []message.Message) []message.Message {
	// First pass: filter out canceled messages
	var cleanedMessages []message.Message
	for _, msg := range messages {
		if msg.FinishReason() != "canceled" || len(msg.ToolCalls()) > 0 {
			// if there are toolCalls this means we want to return it to the LLM telling it that those tools have been
			// cancelled
			cleanedMessages = append(cleanedMessages, msg)
		}
	}

	// Second pass: filter out tool messages without a corresponding tool call
	var result []message.Message
	toolMessageIDs := make(map[string]bool)

	for _, msg := range cleanedMessages {
		if msg.Role == message.Assistant {
			for _, toolCall := range msg.ToolCalls() {
				toolMessageIDs[toolCall.ID] = true // Mark as referenced
			}
		}
	}

	// Keep only messages that aren't unreferenced tool messages
	for _, msg := range cleanedMessages {
		if msg.Role == message.Tool {
			for _, toolCall := range msg.ToolResults() {
				if referenced, exists := toolMessageIDs[toolCall.ToolCallID]; exists && referenced {
					result = append(result, msg)
				}
			}
		} else {
			result = append(result, msg)
		}
	}
	return result
}
