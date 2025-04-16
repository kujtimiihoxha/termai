package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/kujtimiihoxha/opencode/internal/config"
	"github.com/kujtimiihoxha/opencode/internal/llm/tools"
	"github.com/kujtimiihoxha/opencode/internal/logging"
	"github.com/kujtimiihoxha/opencode/internal/message"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type geminiOptions struct {
	disableCache bool
}

type GeminiOption func(*geminiOptions)

type geminiClient struct {
	providerOptions providerClientOptions
	options         geminiOptions
	client          *genai.Client
}

type GeminiClient ProviderClient

func newGeminiClient(opts providerClientOptions) GeminiClient {
	geminiOpts := geminiOptions{}
	for _, o := range opts.geminiOptions {
		o(&geminiOpts)
	}

	client, err := genai.NewClient(context.Background(), option.WithAPIKey(opts.apiKey))
	if err != nil {
		logging.Error("Failed to create Gemini client", "error", err)
		return nil
	}

	return &geminiClient{
		providerOptions: opts,
		options:         geminiOpts,
		client:          client,
	}
}

func (g *geminiClient) convertMessages(messages []message.Message) []*genai.Content {
	var history []*genai.Content

	// Add system message first
	history = append(history, &genai.Content{
		Parts: []genai.Part{genai.Text(g.providerOptions.systemMessage)},
		Role:  "user",
	})

	// Add a system response to acknowledge the system message
	history = append(history, &genai.Content{
		Parts: []genai.Part{genai.Text("I'll help you with that.")},
		Role:  "model",
	})

	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			history = append(history, &genai.Content{
				Parts: []genai.Part{genai.Text(msg.Content().String())},
				Role:  "user",
			})

		case message.Assistant:
			content := &genai.Content{
				Role:  "model",
				Parts: []genai.Part{},
			}

			if msg.Content().String() != "" {
				content.Parts = append(content.Parts, genai.Text(msg.Content().String()))
			}

			if len(msg.ToolCalls()) > 0 {
				for _, call := range msg.ToolCalls() {
					args, _ := parseJsonToMap(call.Input)
					content.Parts = append(content.Parts, genai.FunctionCall{
						Name: call.Name,
						Args: args,
					})
				}
			}

			history = append(history, content)

		case message.Tool:
			for _, result := range msg.ToolResults() {
				response := map[string]interface{}{"result": result.Content}
				parsed, err := parseJsonToMap(result.Content)
				if err == nil {
					response = parsed
				}

				var toolCall message.ToolCall
				for _, m := range messages {
					if m.Role == message.Assistant {
						for _, call := range m.ToolCalls() {
							if call.ID == result.ToolCallID {
								toolCall = call
								break
							}
						}
					}
				}

				history = append(history, &genai.Content{
					Parts: []genai.Part{genai.FunctionResponse{
						Name:     toolCall.Name,
						Response: response,
					}},
					Role: "function",
				})
			}
		}
	}

	return history
}

func (g *geminiClient) convertTools(tools []tools.BaseTool) []*genai.Tool {
	geminiTools := make([]*genai.Tool, 0, len(tools))

	for _, tool := range tools {
		info := tool.Info()
		declaration := &genai.FunctionDeclaration{
			Name:        info.Name,
			Description: info.Description,
			Parameters: &genai.Schema{
				Type:       genai.TypeObject,
				Properties: convertSchemaProperties(info.Parameters),
				Required:   info.Required,
			},
		}

		geminiTools = append(geminiTools, &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{declaration},
		})
	}

	return geminiTools
}

func (g *geminiClient) finishReason(reason genai.FinishReason) message.FinishReason {
	reasonStr := reason.String()
	switch {
	case reasonStr == "STOP":
		return message.FinishReasonEndTurn
	case reasonStr == "MAX_TOKENS":
		return message.FinishReasonMaxTokens
	case strings.Contains(reasonStr, "FUNCTION") || strings.Contains(reasonStr, "TOOL"):
		return message.FinishReasonToolUse
	default:
		return message.FinishReasonUnknown
	}
}

func (g *geminiClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	model := g.client.GenerativeModel(g.providerOptions.model.APIModel)
	model.SetMaxOutputTokens(int32(g.providerOptions.maxTokens))

	// Convert tools
	if len(tools) > 0 {
		model.Tools = g.convertTools(tools)
	}

	// Convert messages
	geminiMessages := g.convertMessages(messages)

	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(geminiMessages)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}

	attempts := 0
	for {
		attempts++
		chat := model.StartChat()
		chat.History = geminiMessages[:len(geminiMessages)-1] // All but last message

		lastMsg := geminiMessages[len(geminiMessages)-1]
		var lastText string
		for _, part := range lastMsg.Parts {
			if text, ok := part.(genai.Text); ok {
				lastText = string(text)
				break
			}
		}

		resp, err := chat.SendMessage(ctx, genai.Text(lastText))
		// If there is an error we are going to see if we can retry the call
		if err != nil {
			retry, after, retryErr := g.shouldRetry(attempts, err)
			if retryErr != nil {
				return nil, retryErr
			}
			if retry {
				logging.WarnPersist("Retrying due to rate limit... attempt %d of %d", logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			return nil, retryErr
		}

		content := ""
		var toolCalls []message.ToolCall

		if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
			for _, part := range resp.Candidates[0].Content.Parts {
				switch p := part.(type) {
				case genai.Text:
					content = string(p)
				case genai.FunctionCall:
					id := "call_" + uuid.New().String()
					args, _ := json.Marshal(p.Args)
					toolCalls = append(toolCalls, message.ToolCall{
						ID:    id,
						Name:  p.Name,
						Input: string(args),
						Type:  "function",
					})
				}
			}
		}

		return &ProviderResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        g.usage(resp),
			FinishReason: g.finishReason(resp.Candidates[0].FinishReason),
		}, nil
	}
}

func (g *geminiClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	model := g.client.GenerativeModel(g.providerOptions.model.APIModel)
	model.SetMaxOutputTokens(int32(g.providerOptions.maxTokens))

	// Convert tools
	if len(tools) > 0 {
		model.Tools = g.convertTools(tools)
	}

	// Convert messages
	geminiMessages := g.convertMessages(messages)

	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(geminiMessages)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}

	attempts := 0
	eventChan := make(chan ProviderEvent)

	go func() {
		defer close(eventChan)

		for {
			attempts++
			chat := model.StartChat()
			chat.History = geminiMessages[:len(geminiMessages)-1] // All but last message

			lastMsg := geminiMessages[len(geminiMessages)-1]
			var lastText string
			for _, part := range lastMsg.Parts {
				if text, ok := part.(genai.Text); ok {
					lastText = string(text)
					break
				}
			}

			iter := chat.SendMessageStream(ctx, genai.Text(lastText))

			currentContent := ""
			toolCalls := []message.ToolCall{}
			var finalResp *genai.GenerateContentResponse

			eventChan <- ProviderEvent{Type: EventContentStart}

			for {
				resp, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					retry, after, retryErr := g.shouldRetry(attempts, err)
					if retryErr != nil {
						eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
						return
					}
					if retry {
						logging.WarnPersist("Retrying due to rate limit... attempt %d of %d", logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
						select {
						case <-ctx.Done():
							if ctx.Err() != nil {
								eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
							}

							return
						case <-time.After(time.Duration(after) * time.Millisecond):
							break
						}
					} else {
						eventChan <- ProviderEvent{Type: EventError, Error: err}
						return
					}
				}

				finalResp = resp

				if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
					for _, part := range resp.Candidates[0].Content.Parts {
						switch p := part.(type) {
						case genai.Text:
							newText := string(p)
							delta := newText[len(currentContent):]
							if delta != "" {
								eventChan <- ProviderEvent{
									Type:    EventContentDelta,
									Content: delta,
								}
								currentContent = newText
							}
						case genai.FunctionCall:
							id := "call_" + uuid.New().String()
							args, _ := json.Marshal(p.Args)
							newCall := message.ToolCall{
								ID:    id,
								Name:  p.Name,
								Input: string(args),
								Type:  "function",
							}

							isNew := true
							for _, existing := range toolCalls {
								if existing.Name == newCall.Name && existing.Input == newCall.Input {
									isNew = false
									break
								}
							}

							if isNew {
								toolCalls = append(toolCalls, newCall)
							}
						}
					}
				}
			}

			eventChan <- ProviderEvent{Type: EventContentStop}

			if finalResp != nil {
				eventChan <- ProviderEvent{
					Type: EventComplete,
					Response: &ProviderResponse{
						Content:      currentContent,
						ToolCalls:    toolCalls,
						Usage:        g.usage(finalResp),
						FinishReason: g.finishReason(finalResp.Candidates[0].FinishReason),
					},
				}
				return
			}

			// If we get here, we need to retry
			if attempts > maxRetries {
				eventChan <- ProviderEvent{
					Type:  EventError,
					Error: fmt.Errorf("maximum retry attempts reached: %d retries", maxRetries),
				}
				return
			}

			// Wait before retrying
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
				}
				return
			case <-time.After(time.Duration(2000*(1<<(attempts-1))) * time.Millisecond):
				continue
			}
		}
	}()

	return eventChan
}

func (g *geminiClient) shouldRetry(attempts int, err error) (bool, int64, error) {
	// Check if error is a rate limit error
	if attempts > maxRetries {
		return false, 0, fmt.Errorf("maximum retry attempts reached for rate limit: %d retries", maxRetries)
	}

	// Gemini doesn't have a standard error type we can check against
	// So we'll check the error message for rate limit indicators
	if errors.Is(err, io.EOF) {
		return false, 0, err
	}

	errMsg := err.Error()
	isRateLimit := false

	// Check for common rate limit error messages
	if contains(errMsg, "rate limit", "quota exceeded", "too many requests") {
		isRateLimit = true
	}

	if !isRateLimit {
		return false, 0, err
	}

	// Calculate backoff with jitter
	backoffMs := 2000 * (1 << (attempts - 1))
	jitterMs := int(float64(backoffMs) * 0.2)
	retryMs := backoffMs + jitterMs

	return true, int64(retryMs), nil
}

func (g *geminiClient) toolCalls(resp *genai.GenerateContentResponse) []message.ToolCall {
	var toolCalls []message.ToolCall

	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if funcCall, ok := part.(genai.FunctionCall); ok {
				id := "call_" + uuid.New().String()
				args, _ := json.Marshal(funcCall.Args)
				toolCalls = append(toolCalls, message.ToolCall{
					ID:    id,
					Name:  funcCall.Name,
					Input: string(args),
					Type:  "function",
				})
			}
		}
	}

	return toolCalls
}

func (g *geminiClient) usage(resp *genai.GenerateContentResponse) TokenUsage {
	if resp == nil || resp.UsageMetadata == nil {
		return TokenUsage{}
	}

	return TokenUsage{
		InputTokens:         int64(resp.UsageMetadata.PromptTokenCount),
		OutputTokens:        int64(resp.UsageMetadata.CandidatesTokenCount),
		CacheCreationTokens: 0, // Not directly provided by Gemini
		CacheReadTokens:     int64(resp.UsageMetadata.CachedContentTokenCount),
	}
}

func WithGeminiDisableCache() GeminiOption {
	return func(options *geminiOptions) {
		options.disableCache = true
	}
}

// Helper functions
func parseJsonToMap(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	return result, err
}

func convertSchemaProperties(parameters map[string]interface{}) map[string]*genai.Schema {
	properties := make(map[string]*genai.Schema)

	for name, param := range parameters {
		properties[name] = convertToSchema(param)
	}

	return properties
}

func convertToSchema(param interface{}) *genai.Schema {
	schema := &genai.Schema{Type: genai.TypeString}

	paramMap, ok := param.(map[string]interface{})
	if !ok {
		return schema
	}

	if desc, ok := paramMap["description"].(string); ok {
		schema.Description = desc
	}

	typeVal, hasType := paramMap["type"]
	if !hasType {
		return schema
	}

	typeStr, ok := typeVal.(string)
	if !ok {
		return schema
	}

	schema.Type = mapJSONTypeToGenAI(typeStr)

	switch typeStr {
	case "array":
		schema.Items = processArrayItems(paramMap)
	case "object":
		if props, ok := paramMap["properties"].(map[string]interface{}); ok {
			schema.Properties = convertSchemaProperties(props)
		}
	}

	return schema
}

func processArrayItems(paramMap map[string]interface{}) *genai.Schema {
	items, ok := paramMap["items"].(map[string]interface{})
	if !ok {
		return nil
	}

	return convertToSchema(items)
}

func mapJSONTypeToGenAI(jsonType string) genai.Type {
	switch jsonType {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeString // Default to string for unknown types
	}
}

func contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

