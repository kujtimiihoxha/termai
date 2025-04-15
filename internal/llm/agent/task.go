package agent

import (
	"context"
	"errors"

	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/lsp"
	"github.com/kujtimiihoxha/termai/internal/message"
	"github.com/kujtimiihoxha/termai/internal/session"
)

type taskAgent struct {
	Service
}

func NewTaskAgent(messages message.Service, sessions session.Service, lspClients map[string]*lsp.Client) (Service, error) {
	model, ok := models.SupportedModels[config.Get().Model.Coder]
	if !ok {
		return nil, errors.New("model not supported")
	}

	ctx := context.Background()

	agent, err := NewAgent(
		ctx,
		sessions,
		messages,
		model,
		[]tools.BaseTool{
			tools.NewGlobTool(),
			tools.NewGrepTool(),
			tools.NewLsTool(),
			tools.NewSourcegraphTool(),
			tools.NewViewTool(lspClients),
		},
	)
	if err != nil {
		return nil, err
	}

	return &taskAgent{
		agent,
	}, nil
}
