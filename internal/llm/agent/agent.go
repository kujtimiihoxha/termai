package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/llm/prompt"
	"github.com/kujtimiihoxha/termai/internal/llm/provider"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/logging"
	"github.com/kujtimiihoxha/termai/internal/message"
	"github.com/kujtimiihoxha/termai/internal/session"
)

// Common errors
var (
	ErrProviderNotEnabled = errors.New("provider is not enabled")
	ErrRequestCancelled   = errors.New("request cancelled by user")
	ErrSessionBusy        = errors.New("session is currently processing another request")
)

// Service defines the interface for generating responses
type Service interface {
	Generate(ctx context.Context, sessionID string, content string) error
	Cancel(sessionID string) error
}

type agent struct {
	sessions       session.Service
	messages       message.Service
	model          models.Model
	tools          []tools.BaseTool
	agent          provider.Provider
	titleGenerator provider.Provider
	activeRequests sync.Map // map[sessionID]context.CancelFunc
}

// NewAgent creates a new agent instance with the given model and tools
func NewAgent(ctx context.Context, sessions session.Service, messages message.Service, model models.Model, tools []tools.BaseTool) (Service, error) {
	agentProvider, titleGenerator, err := getAgentProviders(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	return &agent{
		model:          model,
		tools:          tools,
		sessions:       sessions,
		messages:       messages,
		agent:          agentProvider,
		titleGenerator: titleGenerator,
		activeRequests: sync.Map{},
	}, nil
}

// Cancel cancels an active request by session ID
func (a *agent) Cancel(sessionID string) error {
	if cancelFunc, exists := a.activeRequests.LoadAndDelete(sessionID); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			logging.InfoPersist(fmt.Sprintf("Request cancellation initiated for session: %s", sessionID))
			cancel()
			return nil
		}
	}
	return errors.New("no active request found for this session")
}

// Generate starts the generation process
func (a *agent) Generate(ctx context.Context, sessionID string, content string) error {
	// Check if this session already has an active request
	if _, busy := a.activeRequests.Load(sessionID); busy {
		return ErrSessionBusy
	}

	// Create a cancellable context
	genCtx, cancel := context.WithCancel(ctx)

	// Store cancel function to allow user cancellation
	a.activeRequests.Store(sessionID, cancel)

	// Launch the generation in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logging.ErrorPersist(fmt.Sprintf("Panic in Generate: %v", r))
			}
		}()
		defer a.activeRequests.Delete(sessionID)
		defer cancel()

		if err := a.generate(genCtx, sessionID, content); err != nil {
			if !errors.Is(err, ErrRequestCancelled) && !errors.Is(err, context.Canceled) {
				// Log the error (avoid logging cancellations as they're expected)
				logging.ErrorPersist(fmt.Sprintf("Generation error for session %s: %v", sessionID, err))

				// You may want to create an error message in the chat
				bgCtx := context.Background()
				errorMsg := fmt.Sprintf("Sorry, an error occurred: %v", err)
				_, createErr := a.messages.Create(bgCtx, sessionID, message.CreateMessageParams{
					Role: message.System,
					Parts: []message.ContentPart{
						message.TextContent{
							Text: errorMsg,
						},
					},
				})
				if createErr != nil {
					logging.ErrorPersist(fmt.Sprintf("Failed to create error message: %v", createErr))
				}
			}
		}
	}()

	return nil
}

// IsSessionBusy checks if a session currently has an active request
func (a *agent) IsSessionBusy(sessionID string) bool {
	_, busy := a.activeRequests.Load(sessionID)
	return busy
} // handleTitleGeneration asynchronously generates a title for new sessions
func (a *agent) handleTitleGeneration(ctx context.Context, sessionID, content string) {
	response, err := a.titleGenerator.SendMessages(
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
		logging.ErrorPersist(fmt.Sprintf("Failed to generate title: %v", err))
		return
	}

	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		logging.ErrorPersist(fmt.Sprintf("Failed to get session: %v", err))
		return
	}

	if response.Content != "" {
		session.Title = strings.TrimSpace(response.Content)
		session.Title = strings.ReplaceAll(session.Title, "\n", " ")
		if _, err := a.sessions.Save(ctx, session); err != nil {
			logging.ErrorPersist(fmt.Sprintf("Failed to save session title: %v", err))
		}
	}
}

// TrackUsage updates token usage statistics for the session
func (a *agent) TrackUsage(ctx context.Context, sessionID string, model models.Model, usage provider.TokenUsage) error {
	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	cost := model.CostPer1MInCached/1e6*float64(usage.CacheCreationTokens) +
		model.CostPer1MOutCached/1e6*float64(usage.CacheReadTokens) +
		model.CostPer1MIn/1e6*float64(usage.InputTokens) +
		model.CostPer1MOut/1e6*float64(usage.OutputTokens)

	session.Cost += cost
	session.CompletionTokens += usage.OutputTokens
	session.PromptTokens += usage.InputTokens

	_, err = a.sessions.Save(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

// processEvent handles different types of events during generation
func (a *agent) processEvent(
	ctx context.Context,
	sessionID string,
	assistantMsg *message.Message,
	event provider.ProviderEvent,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	switch event.Type {
	case provider.EventThinkingDelta:
		assistantMsg.AppendReasoningContent(event.Content)
		return a.messages.Update(ctx, *assistantMsg)
	case provider.EventContentDelta:
		assistantMsg.AppendContent(event.Content)
		return a.messages.Update(ctx, *assistantMsg)
	case provider.EventError:
		if errors.Is(event.Error, context.Canceled) {
			logging.InfoPersist(fmt.Sprintf("Event processing canceled for session: %s", sessionID))
			return context.Canceled
		}
		logging.ErrorPersist(event.Error.Error())
		return event.Error
	case provider.EventWarning:
		logging.WarnPersist(event.Info)
	case provider.EventInfo:
		logging.InfoPersist(event.Info)
	case provider.EventComplete:
		assistantMsg.SetToolCalls(event.Response.ToolCalls)
		assistantMsg.AddFinish(event.Response.FinishReason)
		if err := a.messages.Update(ctx, *assistantMsg); err != nil {
			return fmt.Errorf("failed to update message: %w", err)
		}
		return a.TrackUsage(ctx, sessionID, a.model, event.Response.Usage)
	}

	return nil
}

// ExecuteTools runs all tool calls sequentially and returns the results
func (a *agent) ExecuteTools(ctx context.Context, toolCalls []message.ToolCall, tls []tools.BaseTool) ([]message.ToolResult, error) {
	toolResults := make([]message.ToolResult, len(toolCalls))

	// Create a child context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Check if already canceled before starting any execution
	if ctx.Err() != nil {
		// Mark all tools as canceled
		for i, toolCall := range toolCalls {
			toolResults[i] = message.ToolResult{
				ToolCallID: toolCall.ID,
				Content:    "Tool execution canceled by user",
				IsError:    true,
			}
		}
		return toolResults, ctx.Err()
	}

	for i, toolCall := range toolCalls {
		// Check for cancellation before executing each tool
		select {
		case <-ctx.Done():
			// Mark this and all remaining tools as canceled
			for j := i; j < len(toolCalls); j++ {
				toolResults[j] = message.ToolResult{
					ToolCallID: toolCalls[j].ID,
					Content:    "Tool execution canceled by user",
					IsError:    true,
				}
			}
			return toolResults, ctx.Err()
		default:
			// Continue processing
		}

		response := ""
		isError := false
		found := false

		// Find and execute the appropriate tool
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
						response = "Tool execution canceled by user"
					} else {
						response = fmt.Sprintf("Error running tool: %s", toolErr)
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
			response = fmt.Sprintf("Tool not found: %s", toolCall.Name)
			isError = true
		}

		toolResults[i] = message.ToolResult{
			ToolCallID: toolCall.ID,
			Content:    response,
			IsError:    isError,
		}
	}

	return toolResults, nil
}

// handleToolExecution processes tool calls and creates tool result messages
func (a *agent) handleToolExecution(
	ctx context.Context,
	assistantMsg message.Message,
) (*message.Message, error) {
	select {
	case <-ctx.Done():
		// If cancelled, create tool results that indicate cancellation
		if len(assistantMsg.ToolCalls()) > 0 {
			toolResults := make([]message.ToolResult, 0, len(assistantMsg.ToolCalls()))
			for _, tc := range assistantMsg.ToolCalls() {
				toolResults = append(toolResults, message.ToolResult{
					ToolCallID: tc.ID,
					Content:    "Tool execution canceled by user",
					IsError:    true,
				})
			}

			// Use background context to ensure the message is created even if original context is cancelled
			bgCtx := context.Background()
			parts := make([]message.ContentPart, 0)
			for _, toolResult := range toolResults {
				parts = append(parts, toolResult)
			}
			msg, err := a.messages.Create(bgCtx, assistantMsg.SessionID, message.CreateMessageParams{
				Role:  message.Tool,
				Parts: parts,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create cancelled tool message: %w", err)
			}
			return &msg, ctx.Err()
		}
		return nil, ctx.Err()
	default:
		// Continue processing
	}

	if len(assistantMsg.ToolCalls()) == 0 {
		return nil, nil
	}

	toolResults, err := a.ExecuteTools(ctx, assistantMsg.ToolCalls(), a.tools)
	if err != nil {
		// If error is from cancellation, still return the partial results we have
		if errors.Is(err, context.Canceled) {
			// Use background context to ensure the message is created even if original context is cancelled
			bgCtx := context.Background()
			parts := make([]message.ContentPart, 0)
			for _, toolResult := range toolResults {
				parts = append(parts, toolResult)
			}

			msg, createErr := a.messages.Create(bgCtx, assistantMsg.SessionID, message.CreateMessageParams{
				Role:  message.Tool,
				Parts: parts,
			})
			if createErr != nil {
				logging.ErrorPersist(fmt.Sprintf("Failed to create tool message after cancellation: %v", createErr))
				return nil, err
			}
			return &msg, err
		}
		return nil, err
	}

	parts := make([]message.ContentPart, 0, len(toolResults))
	for _, toolResult := range toolResults {
		parts = append(parts, toolResult)
	}

	msg, err := a.messages.Create(ctx, assistantMsg.SessionID, message.CreateMessageParams{
		Role:  message.Tool,
		Parts: parts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tool message: %w", err)
	}

	return &msg, nil
}

// generate handles the main generation workflow
func (a *agent) generate(ctx context.Context, sessionID string, content string) error {
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)

	// Handle context cancellation at any point
	if err := ctx.Err(); err != nil {
		return ErrRequestCancelled
	}

	messages, err := a.messages.List(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if len(messages) == 0 {
		titleCtx := context.Background()
		go a.handleTitleGeneration(titleCtx, sessionID, content)
	}

	userMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{
				Text: content,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create user message: %w", err)
	}

	messages = append(messages, userMsg)

	for {
		// Check for cancellation before each iteration
		select {
		case <-ctx.Done():
			return ErrRequestCancelled
		default:
			// Continue processing
		}

		eventChan, err := a.agent.StreamResponse(ctx, messages, a.tools)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return ErrRequestCancelled
			}
			return fmt.Errorf("failed to stream response: %w", err)
		}

		assistantMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
			Role:  message.Assistant,
			Parts: []message.ContentPart{},
			Model: a.model.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to create assistant message: %w", err)
		}

		ctx = context.WithValue(ctx, tools.MessageIDContextKey, assistantMsg.ID)

		// Process events from the LLM provider
		for event := range eventChan {
			if err := a.processEvent(ctx, sessionID, &assistantMsg, event); err != nil {
				if errors.Is(err, context.Canceled) {
					// Mark as canceled but don't create separate message
					assistantMsg.AddFinish("canceled")
					_ = a.messages.Update(context.Background(), assistantMsg)
					return ErrRequestCancelled
				}
				assistantMsg.AddFinish("error:" + err.Error())
				_ = a.messages.Update(ctx, assistantMsg)
				return fmt.Errorf("event processing error: %w", err)
			}

			// Check for cancellation during event processing
			select {
			case <-ctx.Done():
				// Mark as canceled
				assistantMsg.AddFinish("canceled")
				_ = a.messages.Update(context.Background(), assistantMsg)
				return ErrRequestCancelled
			default:
			}
		}

		// Check for cancellation before tool execution
		select {
		case <-ctx.Done():
			assistantMsg.AddFinish("canceled_by_user")
			_ = a.messages.Update(context.Background(), assistantMsg)
			return ErrRequestCancelled
		default:
		}

		// Execute any tool calls
		toolMsg, err := a.handleToolExecution(ctx, assistantMsg)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				assistantMsg.AddFinish("canceled_by_user")
				_ = a.messages.Update(context.Background(), assistantMsg)
				return ErrRequestCancelled
			}
			return fmt.Errorf("tool execution error: %w", err)
		}

		if err := a.messages.Update(ctx, assistantMsg); err != nil {
			return fmt.Errorf("failed to update assistant message: %w", err)
		}

		// If no tool calls, we're done
		if len(assistantMsg.ToolCalls()) == 0 {
			break
		}

		// Add messages for next iteration
		messages = append(messages, assistantMsg)
		if toolMsg != nil {
			messages = append(messages, *toolMsg)
		}

		// Check for cancellation after tool execution
		select {
		case <-ctx.Done():
			return ErrRequestCancelled
		default:
		}
	}

	return nil
}

// getAgentProviders initializes the LLM providers based on the chosen model
func getAgentProviders(ctx context.Context, model models.Model) (provider.Provider, provider.Provider, error) {
	maxTokens := config.Get().Model.CoderMaxTokens

	providerConfig, ok := config.Get().Providers[model.Provider]
	if !ok || providerConfig.Disabled {
		return nil, nil, ErrProviderNotEnabled
	}

	var agentProvider provider.Provider
	var titleGenerator provider.Provider
	var err error

	switch model.Provider {
	case models.ProviderOpenAI:
		agentProvider, err = provider.NewOpenAIProvider(
			provider.WithOpenAISystemMessage(
				prompt.CoderOpenAISystemPrompt(),
			),
			provider.WithOpenAIMaxTokens(maxTokens),
			provider.WithOpenAIModel(model),
			provider.WithOpenAIKey(providerConfig.APIKey),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create OpenAI agent provider: %w", err)
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
			return nil, nil, fmt.Errorf("failed to create OpenAI title generator: %w", err)
		}

	case models.ProviderAnthropic:
		agentProvider, err = provider.NewAnthropicProvider(
			provider.WithAnthropicSystemMessage(
				prompt.CoderAnthropicSystemPrompt(),
			),
			provider.WithAnthropicMaxTokens(maxTokens),
			provider.WithAnthropicKey(providerConfig.APIKey),
			provider.WithAnthropicModel(model),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Anthropic agent provider: %w", err)
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
			return nil, nil, fmt.Errorf("failed to create Anthropic title generator: %w", err)
		}

	case models.ProviderGemini:
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
			return nil, nil, fmt.Errorf("failed to create Gemini agent provider: %w", err)
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
			return nil, nil, fmt.Errorf("failed to create Gemini title generator: %w", err)
		}

	case models.ProviderGROQ:
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
			return nil, nil, fmt.Errorf("failed to create GROQ agent provider: %w", err)
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
			return nil, nil, fmt.Errorf("failed to create GROQ title generator: %w", err)
		}

	case models.ProviderBedrock:
		agentProvider, err = provider.NewBedrockProvider(
			provider.WithBedrockSystemMessage(
				prompt.CoderAnthropicSystemPrompt(),
			),
			provider.WithBedrockMaxTokens(maxTokens),
			provider.WithBedrockModel(model),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Bedrock agent provider: %w", err)
		}

		titleGenerator, err = provider.NewBedrockProvider(
			provider.WithBedrockSystemMessage(
				prompt.TitlePrompt(),
			),
			provider.WithBedrockMaxTokens(80),
			provider.WithBedrockModel(model),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Bedrock title generator: %w", err)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported provider: %s", model.Provider)
	}

	return agentProvider, titleGenerator, nil
}
