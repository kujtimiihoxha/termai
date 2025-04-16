package prompt

import (
	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
)

func GetAgentPrompt(agentName config.AgentName, provider models.ModelProvider) string {
	switch agentName {
	case config.AgentCoder:
		return CoderPrompt(provider)
	case config.AgentTitle:
		return TitlePrompt(provider)
	case config.AgentTask:
		return TaskPrompt(provider)
	default:
		return "You are a helpful assistant"
	}
}
