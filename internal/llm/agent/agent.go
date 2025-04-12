package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/llm/prompt"
	"github.com/kujtimiihoxha/termai/internal/llm/provider"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/logging"
	"github.com/kujtimiihoxha/termai/internal/message"
)

type Agent interface {
	Generate(ctx context.Context, sessionID string, content string) error
}

type agent struct {
	*app.App
	model          models.Model
	tools          []tools.BaseTool
	agent          provider.Provider
	titleGenerator provider.Provider
}

func (c *agent) handleTitleGeneration(ctx context.Context, sessionID, content string) {
	response, err := c.titleGenerator.SendMessages(
		ctx,
		[]message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{
						Text: content,
					},
				},
			},
		},
		nil,
	)
	if err != nil {
		return
	}

	session, err := c.Sessions.Get(sessionID)
	if err != nil {
		return
	}
	if response.Content != "" {
		session.Title = response.Content
		session.Title = strings.TrimSpace(session.Title)
		session.Title = strings.ReplaceAll(session.Title, "\n", " ")
		c.Sessions.Save(session)
	}
}

func (c *agent) TrackUsage(sessionID string, model models.Model, usage provider.TokenUsage) error {
	session, err := c.Sessions.Get(sessionID)
	if err != nil {
		return err
	}

	cost := model.CostPer1MInCached/1e6*float64(usage.CacheCreationTokens) +
		model.CostPer1MOutCached/1e6*float64(usage.CacheReadTokens) +
		model.CostPer1MIn/1e6*float64(usage.InputTokens) +
		model.CostPer1MOut/1e6*float64(usage.OutputTokens)

	session.Cost += cost
	session.CompletionTokens += usage.OutputTokens
	session.PromptTokens += usage.InputTokens

	_, err = c.Sessions.Save(session)
	return err
}

func (c *agent) processEvent(
	sessionID string,
	assistantMsg *message.Message,
	event provider.ProviderEvent,
) error {
	switch event.Type {
	case provider.EventThinkingDelta:
		assistantMsg.AppendReasoningContent(event.Content)
		return c.Messages.Update(*assistantMsg)
	case provider.EventContentDelta:
		assistantMsg.AppendContent(event.Content)
		return c.Messages.Update(*assistantMsg)
	case provider.EventError:
		if errors.Is(event.Error, context.Canceled) {
			return nil
		}
		logging.ErrorPersist(event.Error.Error())
		return event.Error
	case provider.EventWarning:
		logging.WarnPersist(event.Info)
		return nil
	case provider.EventInfo:
		logging.InfoPersist(event.Info)
	case provider.EventComplete:
		assistantMsg.SetToolCalls(event.Response.ToolCalls)
		assistantMsg.AddFinish(event.Response.FinishReason)
		err := c.Messages.Update(*assistantMsg)
		if err != nil {
			return err
		}
		return c.TrackUsage(sessionID, c.model, event.Response.Usage)
	}

	return nil
}

func (c *agent) ExecuteTools(ctx context.Context, toolCalls []message.ToolCall, tls []tools.BaseTool) ([]message.ToolResult, error) {
	var wg sync.WaitGroup
	toolResults := make([]message.ToolResult, len(toolCalls))
	mutex := &sync.Mutex{}
	errChan := make(chan error, 1)

	// Create a child context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(index int, toolCall message.ToolCall) {
			defer wg.Done()

			// Check if context is already canceled
			select {
			case <-ctx.Done():
				mutex.Lock()
				toolResults[index] = message.ToolResult{
					ToolCallID: toolCall.ID,
					Content:    "Tool execution canceled",
					IsError:    true,
				}
				mutex.Unlock()

				// Send cancellation error to error channel if it's empty
				select {
				case errChan <- ctx.Err():
				default:
				}
				return
			default:
			}

			response := ""
			isError := false
			found := false

			for _, tool := range tls {
				if tool.Info().Name == toolCall.Name {
					found = true
					toolResult, toolErr := tool.Run(ctx, tools.ToolCall{
						ID:    toolCall.ID,
						Name:  toolCall.Name,
						Input: toolCall.Input,
					})

					if toolErr != nil {
						if errors.Is(toolErr, context.Canceled) {
							response = "Tool execution canceled"

							// Send cancellation error to error channel if it's empty
							select {
							case errChan <- ctx.Err():
							default:
							}
						} else {
							response = fmt.Sprintf("error running tool: %s", toolErr)
						}
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
				isError = true
			}

			mutex.Lock()
			defer mutex.Unlock()

			toolResults[index] = message.ToolResult{
				ToolCallID: toolCall.ID,
				Content:    response,
				IsError:    isError,
			}
		}(i, tc)
	}

	// Wait for all goroutines to finish or context to be canceled
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All tools completed successfully
	case err := <-errChan:
		// One of the tools encountered a cancellation
		return toolResults, err
	case <-ctx.Done():
		// Context was canceled externally
		return toolResults, ctx.Err()
	}

	return toolResults, nil
}

func (c *agent) handleToolExecution(
	ctx context.Context,
	assistantMsg message.Message,
) (*message.Message, error) {
	if len(assistantMsg.ToolCalls()) == 0 {
		return nil, nil
	}

	toolResults, err := c.ExecuteTools(ctx, assistantMsg.ToolCalls(), c.tools)
	if err != nil {
		return nil, err
	}
	parts := make([]message.ContentPart, 0)
	for _, toolResult := range toolResults {
		parts = append(parts, toolResult)
	}
	msg, err := c.Messages.Create(assistantMsg.SessionID, message.CreateMessageParams{
		Role:  message.Tool,
		Parts: parts,
	})

	return &msg, err
}

func (c *agent) generate(ctx context.Context, sessionID string, content string) error {
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)
	messages, err := c.Messages.List(sessionID)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		go c.handleTitleGeneration(ctx, sessionID, content)
	}

	userMsg, err := c.Messages.Create(sessionID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{
				Text: content,
			},
		},
	})
	if err != nil {
		return err
	}

	messages = append(messages, userMsg)
	for {
		select {
		case <-ctx.Done():
			assistantMsg, err := c.Messages.Create(sessionID, message.CreateMessageParams{
				Role:  message.Assistant,
				Parts: []message.ContentPart{},
			})
			if err != nil {
				return err
			}
			assistantMsg.AddFinish("canceled")
			c.Messages.Update(assistantMsg)
			return context.Canceled
		default:
			// Continue processing
		}

		eventChan, err := c.agent.StreamResponse(ctx, messages, c.tools)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				assistantMsg, err := c.Messages.Create(sessionID, message.CreateMessageParams{
					Role:  message.Assistant,
					Parts: []message.ContentPart{},
				})
				if err != nil {
					return err
				}
				assistantMsg.AddFinish("canceled")
				c.Messages.Update(assistantMsg)
				return context.Canceled
			}
			return err
		}

		assistantMsg, err := c.Messages.Create(sessionID, message.CreateMessageParams{
			Role:  message.Assistant,
			Parts: []message.ContentPart{},
			Model: c.model.ID,
		})
		if err != nil {
			return err
		}

		ctx = context.WithValue(ctx, tools.MessageIDContextKey, assistantMsg.ID)
		for event := range eventChan {
			err = c.processEvent(sessionID, &assistantMsg, event)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					assistantMsg.AddFinish("canceled")
					c.Messages.Update(assistantMsg)
					return context.Canceled
				}
				assistantMsg.AddFinish("error:" + err.Error())
				c.Messages.Update(assistantMsg)
				return err
			}

			select {
			case <-ctx.Done():
				assistantMsg.AddFinish("canceled")
				c.Messages.Update(assistantMsg)
				return context.Canceled
			default:
			}
		}

		// Check for context cancellation before tool execution
		select {
		case <-ctx.Done():
			assistantMsg.AddFinish("canceled")
			c.Messages.Update(assistantMsg)
			return context.Canceled
		default:
			// Continue processing
		}

		msg, err := c.handleToolExecution(ctx, assistantMsg)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				assistantMsg.AddFinish("canceled")
				c.Messages.Update(assistantMsg)
				return context.Canceled
			}
			return err
		}

		c.Messages.Update(assistantMsg)

		if len(assistantMsg.ToolCalls()) == 0 {
			break
		}

		messages = append(messages, assistantMsg)
		if msg != nil {
			messages = append(messages, *msg)
		}

		// Check for context cancellation after tool execution
		select {
		case <-ctx.Done():
			assistantMsg.AddFinish("canceled")
			c.Messages.Update(assistantMsg)
			return context.Canceled
		default:
			// Continue processing
		}
	}
	return nil
}

func getAgentProviders(ctx context.Context, model models.Model) (provider.Provider, provider.Provider, error) {
	maxTokens := config.Get().Model.CoderMaxTokens

	providerConfig, ok := config.Get().Providers[model.Provider]
	if !ok || !providerConfig.Enabled {
		return nil, nil, errors.New("provider is not enabled")
	}
	var agentProvider provider.Provider
	var titleGenerator provider.Provider

	switch model.Provider {
	case models.ProviderOpenAI:
		var err error
		agentProvider, err = provider.NewOpenAIProvider(
			provider.WithOpenAISystemMessage(
				prompt.CoderOpenAISystemPrompt(),
			),
			provider.WithOpenAIMaxTokens(maxTokens),
			provider.WithOpenAIModel(model),
			provider.WithOpenAIKey(providerConfig.APIKey),
		)
		if err != nil {
			return nil, nil, err
		}
		titleGenerator, err = provider.NewOpenAIProvider(
			provider.WithOpenAISystemMessage(
				prompt.TitlePrompt(),
			),
			provider.WithOpenAIMaxTokens(80),
			provider.WithOpenAIModel(model),
			provider.WithOpenAIKey(providerConfig.APIKey),
		)
		if err != nil {
			return nil, nil, err
		}
	case models.ProviderAnthropic:
		var err error
		agentProvider, err = provider.NewAnthropicProvider(
			provider.WithAnthropicSystemMessage(
				prompt.CoderAnthropicSystemPrompt(),
			),
			provider.WithAnthropicMaxTokens(maxTokens),
			provider.WithAnthropicKey(providerConfig.APIKey),
			provider.WithAnthropicModel(model),
		)
		if err != nil {
			return nil, nil, err
		}
		titleGenerator, err = provider.NewAnthropicProvider(
			provider.WithAnthropicSystemMessage(
				prompt.TitlePrompt(),
			),
			provider.WithAnthropicMaxTokens(80),
			provider.WithAnthropicKey(providerConfig.APIKey),
			provider.WithAnthropicModel(model),
		)
		if err != nil {
			return nil, nil, err
		}

	case models.ProviderGemini:
		var err error
		agentProvider, err = provider.NewGeminiProvider(
			ctx,
			provider.WithGeminiSystemMessage(
				prompt.CoderOpenAISystemPrompt(),
			),
			provider.WithGeminiMaxTokens(int32(maxTokens)),
			provider.WithGeminiKey(providerConfig.APIKey),
			provider.WithGeminiModel(model),
		)
		if err != nil {
			return nil, nil, err
		}
		titleGenerator, err = provider.NewGeminiProvider(
			ctx,
			provider.WithGeminiSystemMessage(
				prompt.TitlePrompt(),
			),
			provider.WithGeminiMaxTokens(80),
			provider.WithGeminiKey(providerConfig.APIKey),
			provider.WithGeminiModel(model),
		)
		if err != nil {
			return nil, nil, err
		}
	case models.ProviderGROQ:
		var err error
		agentProvider, err = provider.NewOpenAIProvider(
			provider.WithOpenAISystemMessage(
				prompt.CoderAnthropicSystemPrompt(),
			),
			provider.WithOpenAIMaxTokens(maxTokens),
			provider.WithOpenAIModel(model),
			provider.WithOpenAIKey(providerConfig.APIKey),
			provider.WithOpenAIBaseURL("https://api.groq.com/openai/v1"),
		)
		if err != nil {
			return nil, nil, err
		}
		titleGenerator, err = provider.NewOpenAIProvider(
			provider.WithOpenAISystemMessage(
				prompt.TitlePrompt(),
			),
			provider.WithOpenAIMaxTokens(80),
			provider.WithOpenAIModel(model),
			provider.WithOpenAIKey(providerConfig.APIKey),
			provider.WithOpenAIBaseURL("https://api.groq.com/openai/v1"),
		)
		if err != nil {
			return nil, nil, err
		}

	case models.ProviderBedrock:
		var err error
		agentProvider, err = provider.NewBedrockProvider(
			provider.WithBedrockSystemMessage(
				prompt.CoderAnthropicSystemPrompt(),
			),
			provider.WithBedrockMaxTokens(maxTokens),
			provider.WithBedrockModel(model),
		)
		if err != nil {
			return nil, nil, err
		}
		titleGenerator, err = provider.NewBedrockProvider(
			provider.WithBedrockSystemMessage(
				prompt.TitlePrompt(),
			),
			provider.WithBedrockMaxTokens(maxTokens),
			provider.WithBedrockModel(model),
		)
		if err != nil {
			return nil, nil, err
		}

	}

	return agentProvider, titleGenerator, nil
}
