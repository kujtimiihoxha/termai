package agent

import (
	"context"
	"errors"

	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
)

type coderAgent struct {
	*agent
}

func (c *coderAgent) setAgentTool(sessionID string) {
	inx := -1
	for i, tool := range c.tools {
		if tool.Info().Name == AgentToolName {
			inx = i
			break
		}
	}
	if inx == -1 {
		c.tools = append(c.tools, NewAgentTool(sessionID, c.App))
	} else {
		c.tools[inx] = NewAgentTool(sessionID, c.App)
	}
}

func (c *coderAgent) Generate(ctx context.Context, sessionID string, content string) error {
	c.setAgentTool(sessionID)
	return c.generate(ctx, sessionID, content)
}

func NewCoderAgent(app *app.App) (Agent, error) {
	model, ok := models.SupportedModels[config.Get().Model.Coder]
	if !ok {
		return nil, errors.New("model not supported")
	}

	ctx := context.Background()
	agentProvider, titleGenerator, err := getAgentProviders(ctx, model)
	if err != nil {
		return nil, err
	}

	otherTools := GetMcpTools(ctx, app.Permissions)
	if len(app.LSPClients) > 0 {
		otherTools = append(otherTools, tools.NewDiagnosticsTool(app.LSPClients))
	}
	return &coderAgent{
		agent: &agent{
			App: app,
			tools: append(
				[]tools.BaseTool{
					tools.NewBashTool(app.Permissions),
					tools.NewEditTool(app.LSPClients, app.Permissions),
					tools.NewFetchTool(app.Permissions),
					tools.NewGlobTool(),
					tools.NewGrepTool(),
					tools.NewLsTool(),
					tools.NewSourcegraphTool(),
					tools.NewViewTool(app.LSPClients),
					tools.NewWriteTool(app.LSPClients, app.Permissions),
				}, otherTools...,
			),
			model:          model,
			agent:          agentProvider,
			titleGenerator: titleGenerator,
		},
	}, nil
}
