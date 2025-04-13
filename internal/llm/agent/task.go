package agent

import (
	"context"
	"errors"

	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/lsp"
)

type taskAgent struct {
	*agent
}

func (c *taskAgent) Generate(ctx context.Context, sessionID string, content string) error {
	return c.generate(ctx, sessionID, content)
}

func NewTaskAgent(lspClients map[string]*lsp.Client) (Service, error) {
	model, ok := models.SupportedModels[config.Get().Model.Coder]
	if !ok {
		return nil, errors.New("model not supported")
	}

	ctx := context.Background()
	agentProvider, titleGenerator, err := getAgentProviders(ctx, model)
	if err != nil {
		return nil, err
	}
	return &taskAgent{
		agent: &agent{
			tools: []tools.BaseTool{
				tools.NewGlobTool(),
				tools.NewGrepTool(),
				tools.NewLsTool(),
				tools.NewSourcegraphTool(),
				tools.NewViewTool(lspClients),
			},
			model:          model,
			agent:          agentProvider,
			titleGenerator: titleGenerator,
		},
	}, nil
}
