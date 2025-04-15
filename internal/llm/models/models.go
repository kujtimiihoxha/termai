package models

type (
	ModelID       string
	ModelProvider string
)

type Model struct {
	ID                 ModelID       `json:"id"`
	Name               string        `json:"name"`
	Provider           ModelProvider `json:"provider"`
	APIModel           string        `json:"api_model"`
	CostPer1MIn        float64       `json:"cost_per_1m_in"`
	CostPer1MOut       float64       `json:"cost_per_1m_out"`
	CostPer1MInCached  float64       `json:"cost_per_1m_in_cached"`
	CostPer1MOutCached float64       `json:"cost_per_1m_out_cached"`
}

// Model IDs
const (
	// Anthropic
	Claude35Sonnet ModelID = "claude-3.5-sonnet"
	Claude3Haiku   ModelID = "claude-3-haiku"
	Claude37Sonnet ModelID = "claude-3.7-sonnet"
	// OpenAI
	GPT41 ModelID = "gpt-4.1"

	// GEMINI
	GEMINI25      ModelID = "gemini-2.5"
	GRMINI20Flash ModelID = "gemini-2.0-flash"

	// GROQ
	QWENQwq ModelID = "qwen-qwq"

	// Bedrock
	BedrockClaude37Sonnet ModelID = "bedrock.claude-3.7-sonnet"
)

const (
	ProviderOpenAI    ModelProvider = "openai"
	ProviderAnthropic ModelProvider = "anthropic"
	ProviderBedrock   ModelProvider = "bedrock"
	ProviderGemini    ModelProvider = "gemini"
	ProviderGROQ      ModelProvider = "groq"
)

var SupportedModels = map[ModelID]Model{
	// Anthropic
	Claude35Sonnet: {
		ID:                 Claude35Sonnet,
		Name:               "Claude 3.5 Sonnet",
		Provider:           ProviderAnthropic,
		APIModel:           "claude-3-5-sonnet-latest",
		CostPer1MIn:        3.0,
		CostPer1MInCached:  3.75,
		CostPer1MOutCached: 0.30,
		CostPer1MOut:       15.0,
	},
	Claude3Haiku: {
		ID:                 Claude3Haiku,
		Name:               "Claude 3 Haiku",
		Provider:           ProviderAnthropic,
		APIModel:           "claude-3-haiku-latest",
		CostPer1MIn:        0.80,
		CostPer1MInCached:  1,
		CostPer1MOutCached: 0.08,
		CostPer1MOut:       4,
	},
	Claude37Sonnet: {
		ID:                 Claude37Sonnet,
		Name:               "Claude 3.7 Sonnet",
		Provider:           ProviderAnthropic,
		APIModel:           "claude-3-7-sonnet-latest",
		CostPer1MIn:        3.0,
		CostPer1MInCached:  3.75,
		CostPer1MOutCached: 0.30,
		CostPer1MOut:       15.0,
	},

	// OpenAI
	GPT41: {
		ID:                 GPT41,
		Name:               "GPT-4.1",
		Provider:           ProviderOpenAI,
		APIModel:           "gpt-4.1",
		CostPer1MIn:        2.00,
		CostPer1MInCached:  0.50,
		CostPer1MOutCached: 0,
		CostPer1MOut:       8.00,
	},

	// GEMINI
	GEMINI25: {
		ID:                 GEMINI25,
		Name:               "Gemini 2.5 Pro",
		Provider:           ProviderGemini,
		APIModel:           "gemini-2.5-pro-exp-03-25",
		CostPer1MIn:        0,
		CostPer1MInCached:  0,
		CostPer1MOutCached: 0,
		CostPer1MOut:       0,
	},

	GRMINI20Flash: {
		ID:                 GRMINI20Flash,
		Name:               "Gemini 2.0 Flash",
		Provider:           ProviderGemini,
		APIModel:           "gemini-2.0-flash",
		CostPer1MIn:        0.1,
		CostPer1MInCached:  0,
		CostPer1MOutCached: 0.025,
		CostPer1MOut:       0.4,
	},

	// GROQ
	QWENQwq: {
		ID:                 QWENQwq,
		Name:               "Qwen Qwq",
		Provider:           ProviderGROQ,
		APIModel:           "qwen-qwq-32b",
		CostPer1MIn:        0,
		CostPer1MInCached:  0,
		CostPer1MOutCached: 0,
		CostPer1MOut:       0,
	},

	// Bedrock
	BedrockClaude37Sonnet: {
		ID:                 BedrockClaude37Sonnet,
		Name:               "Bedrock: Claude 3.7 Sonnet",
		Provider:           ProviderBedrock,
		APIModel:           "anthropic.claude-3-7-sonnet-20250219-v1:0",
		CostPer1MIn:        3.0,
		CostPer1MInCached:  3.75,
		CostPer1MOutCached: 0.30,
		CostPer1MOut:       15.0,
	},
}
