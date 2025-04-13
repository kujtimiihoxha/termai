package agent

import (
	"context"
	"errors"

	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/lsp"
	"github.com/kujtimiihoxha/termai/internal/message"
	"github.com/kujtimiihoxha/termai/internal/permission"
	"github.com/kujtimiihoxha/termai/internal/session"
)

type coderAgent struct {
	Service
}

func NewCoderAgent(
	permissions permission.Service,
	sessions session.Service,
	messages message.Service,
	lspClients map[string]*lsp.Client,
) (Service, error) {
	model, ok := models.SupportedModels[config.Get().Model.Coder]
	if !ok {
		return nil, errors.New("model not supported")
	}

	ctx := context.Background()
	otherTools := GetMcpTools(ctx, permissions)
	if len(lspClients) > 0 {
		otherTools = append(otherTools, tools.NewDiagnosticsTool(lspClients))
	}
	agent, err := NewAgent(
		ctx,
		sessions,
		messages,
		model,
		append(
			[]tools.BaseTool{
				tools.NewBashTool(permissions),
				tools.NewEditTool(lspClients, permissions),
				tools.NewFetchTool(permissions),
				tools.NewGlobTool(),
				tools.NewGrepTool(),
				tools.NewLsTool(),
				tools.NewSourcegraphTool(),
				tools.NewViewTool(lspClients),
				tools.NewWriteTool(lspClients, permissions),
				NewAgentTool(sessions, messages),
			}, otherTools...,
		),
	)
	if err != nil {
		return nil, err
	}

	return &coderAgent{
		agent,
	}, nil
}
