package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/message"
)

type anthropicProvider struct {
	client        anthropic.Client
	model         models.Model
	maxTokens     int64
	apiKey        string
	systemMessage string
}

type AnthropicOption func(*anthropicProvider)

func WithAnthropicSystemMessage(message string) AnthropicOption {
	return func(a *anthropicProvider) {
		a.systemMessage = message
	}
}

func WithAnthropicMaxTokens(maxTokens int64) AnthropicOption {
	return func(a *anthropicProvider) {
		a.maxTokens = maxTokens
	}
}

func WithAnthropicModel(model models.Model) AnthropicOption {
	return func(a *anthropicProvider) {
		a.model = model
	}
}

func WithAnthropicKey(apiKey string) AnthropicOption {
	return func(a *anthropicProvider) {
		a.apiKey = apiKey
	}
}

func NewAnthropicProvider(opts ...AnthropicOption) (Provider, error) {
	provider := &anthropicProvider{
		maxTokens: 1024,
	}

	for _, opt := range opts {
		opt(provider)
	}

	if provider.systemMessage == "" {
		return nil, errors.New("system message is required")
	}

	provider.client = anthropic.NewClient(option.WithAPIKey(provider.apiKey))
	return provider, nil
}

func (a *anthropicProvider) SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	anthropicMessages := a.convertToAnthropicMessages(messages)
	anthropicTools := a.convertToAnthropicTools(tools)

	response, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       anthropic.Model(a.model.APIModel),
		MaxTokens:   a.maxTokens,
		Temperature: anthropic.Float(0),
		Messages:    anthropicMessages,
		Tools:       anthropicTools,
		System: []anthropic.TextBlockParam{
			{
				Text: a.systemMessage,
				CacheControl: anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	content := ""
	for _, block := range response.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			content += text.Text
		}
	}

	toolCalls := a.extractToolCalls(response.Content)
	tokenUsage := a.extractTokenUsage(response.Usage)

	return &ProviderResponse{
		Content:   content,
		ToolCalls: toolCalls,
		Usage:     tokenUsage,
	}, nil
}

func (a *anthropicProvider) StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (<-chan ProviderEvent, error) {
	anthropicMessages := a.convertToAnthropicMessages(messages)
	anthropicTools := a.convertToAnthropicTools(tools)

	var thinkingParam anthropic.ThinkingConfigParamUnion
	lastMessage := messages[len(messages)-1]
	temperature := anthropic.Float(0)
	if lastMessage.Role == message.User && strings.Contains(strings.ToLower(lastMessage.Content().String()), "think") {
		thinkingParam = anthropic.ThinkingConfigParamUnion{
			OfThinkingConfigEnabled: &anthropic.ThinkingConfigEnabledParam{
				BudgetTokens: int64(float64(a.maxTokens) * 0.8),
				Type:         "enabled",
			},
		}
		temperature = anthropic.Float(1)
	}

	stream := a.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:       anthropic.Model(a.model.APIModel),
		MaxTokens:   a.maxTokens,
		Temperature: temperature,
		Messages:    anthropicMessages,
		Tools:       anthropicTools,
		Thinking:    thinkingParam,
		System: []anthropic.TextBlockParam{
			{
				Text: a.systemMessage,
				CacheControl: anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				},
			},
		},
	})

	eventChan := make(chan ProviderEvent)

	go func() {
		defer close(eventChan)

		accumulatedMessage := anthropic.Message{}

		for stream.Next() {
			event := stream.Current()
			err := accumulatedMessage.Accumulate(event)
			if err != nil {
				eventChan <- ProviderEvent{Type: EventError, Error: err}
				return
			}

			switch event := event.AsAny().(type) {
			case anthropic.ContentBlockStartEvent:
				eventChan <- ProviderEvent{Type: EventContentStart}

			case anthropic.ContentBlockDeltaEvent:
				if event.Delta.Type == "thinking_delta" && event.Delta.Thinking != "" {
					eventChan <- ProviderEvent{
						Type:     EventThinkingDelta,
						Thinking: event.Delta.Thinking,
					}
				} else if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
					eventChan <- ProviderEvent{
						Type:    EventContentDelta,
						Content: event.Delta.Text,
					}
				}

			case anthropic.ContentBlockStopEvent:
				eventChan <- ProviderEvent{Type: EventContentStop}

			case anthropic.MessageStopEvent:
				content := ""
				for _, block := range accumulatedMessage.Content {
					if text, ok := block.AsAny().(anthropic.TextBlock); ok {
						content += text.Text
					}
				}

				toolCalls := a.extractToolCalls(accumulatedMessage.Content)
				tokenUsage := a.extractTokenUsage(accumulatedMessage.Usage)

				eventChan <- ProviderEvent{
					Type: EventComplete,
					Response: &ProviderResponse{
						Content:      content,
						ToolCalls:    toolCalls,
						Usage:        tokenUsage,
						FinishReason: string(accumulatedMessage.StopReason),
					},
				}
			}
		}

		if stream.Err() != nil {
			eventChan <- ProviderEvent{Type: EventError, Error: stream.Err()}
		}
	}()

	return eventChan, nil
}

func (a *anthropicProvider) extractToolCalls(content []anthropic.ContentBlockUnion) []message.ToolCall {
	var toolCalls []message.ToolCall

	for _, block := range content {
		switch variant := block.AsAny().(type) {
		case anthropic.ToolUseBlock:
			toolCall := message.ToolCall{
				ID:    variant.ID,
				Name:  variant.Name,
				Input: string(variant.Input),
				Type:  string(variant.Type),
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}

func (a *anthropicProvider) extractTokenUsage(usage anthropic.Usage) TokenUsage {
	return TokenUsage{
		InputTokens:         usage.InputTokens,
		OutputTokens:        usage.OutputTokens,
		CacheCreationTokens: usage.CacheCreationInputTokens,
		CacheReadTokens:     usage.CacheReadInputTokens,
	}
}

func (a *anthropicProvider) convertToAnthropicTools(tools []tools.BaseTool) []anthropic.ToolUnionParam {
	anthropicTools := make([]anthropic.ToolUnionParam, len(tools))

	for i, tool := range tools {
		info := tool.Info()
		toolParam := anthropic.ToolParam{
			Name:        info.Name,
			Description: anthropic.String(info.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: info.Parameters,
			},
		}

		if i == len(tools)-1 {
			toolParam.CacheControl = anthropic.CacheControlEphemeralParam{
				Type: "ephemeral",
			}
		}

		anthropicTools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
	}

	return anthropicTools
}

func (a *anthropicProvider) convertToAnthropicMessages(messages []message.Message) []anthropic.MessageParam {
	anthropicMessages := make([]anthropic.MessageParam, 0, len(messages))
	cachedBlocks := 0

	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			content := anthropic.NewTextBlock(msg.Content().String())
			if cachedBlocks < 2 {
				content.OfRequestTextBlock.CacheControl = anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				}
				cachedBlocks++
			}
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(content))

		case message.Assistant:
			blocks := []anthropic.ContentBlockParamUnion{}
			if msg.Content().String() != "" {
				content := anthropic.NewTextBlock(msg.Content().String())
				if cachedBlocks < 2 {
					content.OfRequestTextBlock.CacheControl = anthropic.CacheControlEphemeralParam{
						Type: "ephemeral",
					}
					cachedBlocks++
				}
				blocks = append(blocks, content)
			}

			for _, toolCall := range msg.ToolCalls() {
				var inputMap map[string]any
				err := json.Unmarshal([]byte(toolCall.Input), &inputMap)
				if err != nil {
					continue
				}
				blocks = append(blocks, anthropic.ContentBlockParamOfRequestToolUseBlock(toolCall.ID, inputMap, toolCall.Name))
			}

			// Skip empty assistant messages completely
			if len(blocks) > 0 {
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(blocks...))
			}

		case message.Tool:
			results := make([]anthropic.ContentBlockParamUnion, len(msg.ToolResults()))
			for i, toolResult := range msg.ToolResults() {
				results[i] = anthropic.NewToolResultBlock(toolResult.ToolCallID, toolResult.Content, toolResult.IsError)
			}
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(results...))
		}
	}

	return anthropicMessages
}
