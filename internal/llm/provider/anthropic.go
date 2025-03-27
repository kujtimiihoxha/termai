package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/message"
	"github.com/spf13/viper"
)

type anthropicProvider struct {
	app            *app.App
	maxTokens      int
	systemMessage  string
	anthropicTools []anthropic.ToolUnionParam
	tools          []tools.BaseTool
	client         anthropic.Client
}

// generateSessionTitle creates a title for the session using LLM
func (a *anthropicProvider) generateSessionTitle(ctx context.Context, userMessage string) (string, error) {
	titleSystemMessage := `You will generate a short title based on the first message a user begins a conversation with
- ensure it is not more than 50 characters long
- the title should be a summary of the user's message
- do not use quotes or colons
- the entire text you return will be used as the title`

	// Create a new client message request
	bigModel := viper.GetString("models.big")
	model := models.SupportedModels[models.ModelID(bigModel)].APIModel
	response, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 5000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
		},
		System: []anthropic.TextBlockParam{
			{
				Text: titleSystemMessage,
				CacheControl: anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	// Extract the title from the response
	title := ""
	for _, block := range response.Content {
		title += block.Text
	}

	return title, nil
}

type AnthropicOption func(*anthropicProvider)

// TODO: maybe allow for image messages
func (a *anthropicProvider) NewMessage(sessionID string, content string) error {
	bigModel := viper.GetString("models.big")
	model := models.SupportedModels[models.ModelID(bigModel)]
	messages, err := a.app.Messages.List(sessionID)
	if err != nil {
		return err
	}

	// Set session title if this is the first message - do it asynchronously
	if len(messages) == 0 {
		go func() {
			// Create a separate context for the title generation
			titleCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			title, generateErr := a.generateSessionTitle(titleCtx, content)
			if generateErr != nil {
				log.Printf("failed to generate session title: %v", generateErr)
				return
			}

			session, generateErr := a.app.Sessions.Get(sessionID)
			if generateErr != nil {
				return
			}
			session.Title = title
			_, generateErr = a.app.Sessions.Save(session)
			if generateErr != nil {
				log.Printf("failed to set session title: %v", generateErr)
			}
		}()
		// Continue with message processing without waiting for title generation
	}

	newMessage, err := a.app.Messages.Create(sessionID, message.CreateMessageParams{
		Role:    message.User,
		Content: content,
	})
	if err != nil {
		return err
	}

	messages = append(messages, newMessage)

	anthropicMessages := make([]anthropic.MessageParam, len(messages))
	dmp, _ := json.Marshal(messages)
	log.Println(string(dmp))
	cachedBlocks := 0
	for i, msg := range messages {
		switch msg.Role {
		case message.User:
			content := anthropic.NewTextBlock(msg.Content)
			if cachedBlocks < 2 {
				content.OfRequestTextBlock.CacheControl = anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				}
				cachedBlocks++
			}
			anthropicMessages[i] = anthropic.NewUserMessage(content)
		case message.Assistant:
			blocks := []anthropic.ContentBlockParamUnion{}
			if msg.Content != "" {
				content := anthropic.NewTextBlock(msg.Content)
				if cachedBlocks < 2 {
					content.OfRequestTextBlock.CacheControl = anthropic.CacheControlEphemeralParam{
						Type: "ephemeral",
					}
					cachedBlocks++
				}
				blocks = append(blocks, content)
			}
			for _, toolCall := range msg.ToolCalls {
				var inputMap map[string]any
				err := json.Unmarshal([]byte(toolCall.Input), &inputMap)
				if err != nil {
					return err
				}
				blocks = append(blocks, anthropic.ContentBlockParamOfRequestToolUseBlock(toolCall.ID, inputMap, toolCall.Name))
			}
			anthropicMessages[i] = anthropic.NewAssistantMessage(blocks...)
		case message.Tool:
			results := make([]anthropic.ContentBlockParamUnion, len(msg.ToolResults))
			for i, toolResult := range msg.ToolResults {
				// TODO handle other types of tool results
				results[i] = anthropic.NewToolResultBlock(toolResult.ToolCallID, toolResult.Content, toolResult.IsError)
			}
			anthropicMessages[i] = anthropic.NewUserMessage(results...)
		}
	}
	for {
		stream := a.client.Messages.NewStreaming(context.TODO(), anthropic.MessageNewParams{
			Model:       anthropic.Model(model.APIModel),
			MaxTokens:   5000,
			Temperature: anthropic.Float(0),
			Messages:    anthropicMessages,
			Tools:       a.anthropicTools,
			System: []anthropic.TextBlockParam{
				{
					Text: a.systemMessage,
					CacheControl: anthropic.CacheControlEphemeralParam{
						Type: "ephemeral",
					},
				},
			},
		})

		newAnthropicMessage := anthropic.Message{}
		msg, err := a.app.Messages.Create(sessionID, message.CreateMessageParams{
			Role:    message.Assistant,
			Content: "",
		})
		if err != nil {
			log.Printf("create message: %v", err)
			return err
		}
		for stream.Next() {
			event := stream.Current()
			err = newAnthropicMessage.Accumulate(event)
			if err != nil {
				continue
			}

			switch event := event.AsAny().(type) {
			case anthropic.ContentBlockStartEvent:
			// TODO send out a notification
			case anthropic.ContentBlockDeltaEvent:
				msg.Content += event.Delta.Text
				err = a.app.Messages.Update(msg)
				if err != nil {
					log.Printf("update message: %v", err)
					return err
				}
			case anthropic.ContentBlockStopEvent:
			// TODO send out a notification
			case anthropic.MessageStopEvent:
				// TODO send out a notification
			}
		}

		if stream.Err() != nil {
			log.Printf("error: %v", stream.Err())
			return stream.Err()
		}

		anthropicMessages = append(anthropicMessages, newAnthropicMessage.ToParam())
		anthropicToolResults := []anthropic.ContentBlockParamUnion{}

		// Prepare for parallel tool execution
		var toolCalls []struct {
			toolCall message.ToolCall
			block    anthropic.ToolUseBlock
		}

		// Collect all tool calls first
		for _, block := range newAnthropicMessage.Content {
			switch variant := block.AsAny().(type) {
			case anthropic.ToolUseBlock:
				toolCall := message.ToolCall{
					ID:    variant.ID,
					Name:  variant.Name,
					Input: string(variant.Input),
					Type:  string(variant.Type),
				}

				toolCalls = append(toolCalls, struct {
					toolCall message.ToolCall
					block    anthropic.ToolUseBlock
				}{
					toolCall: toolCall,
					block:    variant,
				})

				msg.ToolCalls = append(msg.ToolCalls, toolCall)
			}
		}

		// Update message with all tool calls
		if len(toolCalls) > 0 {
			err = a.app.Messages.Update(msg)
			if err != nil {
				return err
			}
		}

		cost := model.CostPer1MInCached/1_000_000*float64(newAnthropicMessage.Usage.CacheCreationInputTokens) +
			model.CostPer1MOutCached/1_000_000*float64(newAnthropicMessage.Usage.CacheReadInputTokens) +
			model.CostPer1MIn/1_000_000*float64(newAnthropicMessage.Usage.InputTokens) +
			model.CostPer1MOut/1_000_000*float64(newAnthropicMessage.Usage.OutputTokens)

		session, err := a.app.Sessions.Get(sessionID)
		if err != nil {
			return err
		}
		session.Cost += cost
		session.CompletionTokens += newAnthropicMessage.Usage.OutputTokens
		session.PromptTokens += newAnthropicMessage.Usage.InputTokens

		_, err = a.app.Sessions.Save(session)
		if err != nil {
			return err
		}

		// If there are no tool calls, we're done
		if len(toolCalls) == 0 {
			break
		}

		// Execute tools in parallel
		var wg sync.WaitGroup
		toolResults := make([]message.ToolResult, len(toolCalls))
		resultsMutex := &sync.Mutex{}

		for i, tc := range toolCalls {
			wg.Add(1)
			go func(index int, toolCall message.ToolCall, block anthropic.ToolUseBlock) {
				defer wg.Done()

				response := ""
				isError := false
				found := false

				for _, tool := range a.tools {
					if tool.Info().Name == toolCall.Name {
						found = true
						toolResult, toolErr := tool.Run(context.TODO(), toolCall.Input)
						if toolErr != nil {
							response = fmt.Sprintf("error running tool: %s", toolErr)
							isError = true
						} else {
							response = toolResult.Content
							isError = toolResult.IsError
						}
						break
					}
				}

				if !found {
					response = fmt.Sprintf("tool not found: %s", toolCall.Name)
				}

				resultsMutex.Lock()
				defer resultsMutex.Unlock()

				toolResults[index] = message.ToolResult{
					ToolCallID: toolCall.ID,
					Content:    response,
					IsError:    isError,
				}
				anthropicToolResults = append(anthropicToolResults, anthropic.NewToolResultBlock(block.ID, response, isError))
			}(i, tc.toolCall, tc.block)
		}

		wg.Wait()

		_, err = a.app.Messages.Create(sessionID, message.CreateMessageParams{
			Role:        message.Tool,
			Content:     "",
			ToolResults: toolResults,
		})
		if err != nil {
			return err
		}

		// Add tool results to anthropic messages
		anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(anthropicToolResults...))
	}

	return nil
}

func NewAnthropicProvider(app *app.App, opts ...AnthropicOption) (Provider, error) {
	provider := &anthropicProvider{
		maxTokens: 1024,
		app:       app,
	}
	for _, opt := range opts {
		opt(provider)
	}
	if provider.systemMessage == "" {
		return nil, errors.New("system message is required")
	}

	provider.client = anthropic.NewClient()

	return provider, nil
}

func WithAnthropicSystemMessage(message string) AnthropicOption {
	return func(a *anthropicProvider) {
		a.systemMessage = message
	}
}

func WithAnthropicTools(baseTools []tools.BaseTool) AnthropicOption {
	return func(a *anthropicProvider) {
		tools := make([]anthropic.ToolUnionParam, len(baseTools))
		for i, tool := range baseTools {
			info := tool.Info()
			toolParam := anthropic.ToolParam{
				Name:        info.Name,
				Description: anthropic.String(info.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: info.Parameters,
				},
			}
			if i == len(baseTools)-1 {
				toolParam.CacheControl = anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				}
			}
			tools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
		}
		a.tools = baseTools
		a.anthropicTools = tools
	}
}

func WithAnthropicMaxTokens(maxTokens int) AnthropicOption {
	return func(a *anthropicProvider) {
		a.maxTokens = maxTokens
	}
}
