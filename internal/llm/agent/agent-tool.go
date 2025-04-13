package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/message"
)

type agentTool struct {
	parentSessionID string
	app             *app.App
}

const (
	AgentToolName = "agent"
)

type AgentParams struct {
	Prompt string `json:"prompt"`
}

func (b *agentTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        AgentToolName,
		Description: "Launch a new agent that has access to the following tools: GlobTool, GrepTool, LS, View. When you are searching for a keyword or file and are not confident that you will find the right match on the first try, use the Agent tool to perform the search for you. For example:\n\n- If you are searching for a keyword like \"config\" or \"logger\", or for questions like \"which file does X?\", the Agent tool is strongly recommended\n- If you want to read a specific file path, use the View or GlobTool tool instead of the Agent tool, to find the match more quickly\n- If you are searching for a specific class definition like \"class Foo\", use the GlobTool tool instead, to find the match more quickly\n\nUsage notes:\n1. Launch multiple agents concurrently whenever possible, to maximize performance; to do that, use a single message with multiple tool uses\n2. When the agent is done, it will return a single message back to you. The result returned by the agent is not visible to the user. To show the user the result, you should send a text message back to the user with a concise summary of the result.\n3. Each agent invocation is stateless. You will not be able to send additional messages to the agent, nor will the agent be able to communicate with you outside of its final report. Therefore, your prompt should contain a highly detailed task description for the agent to perform autonomously and you should specify exactly what information the agent should return back to you in its final and only message to you.\n4. The agent's outputs should generally be trusted\n5. IMPORTANT: The agent can not use Bash, Replace, Edit, so can not modify files. If you want to use these tools, use them directly instead of going through the agent.",
		Parameters: map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "The task for the agent to perform",
			},
		},
		Required: []string{"prompt"},
	}
}

func (b *agentTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	var params AgentParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	if params.Prompt == "" {
		return tools.NewTextErrorResponse("prompt is required"), nil
	}

	agent, err := NewTaskAgent(b.app)
	if err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error creating agent: %s", err)), nil
	}

	session, err := b.app.Sessions.CreateTaskSession(ctx, call.ID, b.parentSessionID, "New Agent Session")
	if err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error creating session: %s", err)), nil
	}

	err = agent.Generate(ctx, session.ID, params.Prompt)
	if err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error generating agent: %s", err)), nil
	}

	messages, err := b.app.Messages.List(ctx, session.ID)
	if err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error listing messages: %s", err)), nil
	}
	if len(messages) == 0 {
		return tools.NewTextErrorResponse("no messages found"), nil
	}

	response := messages[len(messages)-1]
	if response.Role != message.Assistant {
		return tools.NewTextErrorResponse("no assistant message found"), nil
	}

	updatedSession, err := b.app.Sessions.Get(ctx, session.ID)
	if err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error: %s", err)), nil
	}
	parentSession, err := b.app.Sessions.Get(ctx, b.parentSessionID)
	if err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error: %s", err)), nil
	}

	parentSession.Cost += updatedSession.Cost
	parentSession.PromptTokens += updatedSession.PromptTokens
	parentSession.CompletionTokens += updatedSession.CompletionTokens

	_, err = b.app.Sessions.Save(ctx, parentSession)
	if err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error: %s", err)), nil
	}
	return tools.NewTextResponse(response.Content().String()), nil
}

func NewAgentTool(parentSessionID string, app *app.App) tools.BaseTool {
	return &agentTool{
		parentSessionID: parentSessionID,
		app:             app,
	}
}
