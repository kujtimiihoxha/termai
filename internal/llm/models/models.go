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

const (
	DefaultBigModel    = Claude37Sonnet
	DefaultLittleModel = Claude37Sonnet
)

// Model IDs
const (
	// Anthropic
	Claude35Sonnet ModelID = "claude-3.5-sonnet"
	Claude3Haiku   ModelID = "claude-3-haiku"
	Claude37Sonnet ModelID = "claude-3.7-sonnet"
)

const (
	ProviderOpenAI    ModelProvider = "openai"
	ProviderAnthropic ModelProvider = "anthropic"
	ProviderGoogle    ModelProvider = "google"
	ProviderXAI       ModelProvider = "xai"
	ProviderDeepSeek  ModelProvider = "deepseek"
	ProviderMeta      ModelProvider = "meta"
	ProviderGroq      ModelProvider = "groq"
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
}

// func GetModel(ctx context.Context, model ModelID) (model.ChatModel, error) {
// 	provider := SupportedModels[model].Provider
// 	log.Printf("Provider: %s", provider)
// 	maxTokens := viper.GetInt("providers.common.max_tokens")
// 	switch provider {
// 	case ProviderOpenAI:
// 		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
// 			APIKey:    viper.GetString("providers.openai.key"),
// 			Model:     string(SupportedModels[model].APIModel),
// 			MaxTokens: &maxTokens,
// 		})
// 	case ProviderAnthropic:
// 		return claude.NewChatModel(ctx, &claude.Config{
// 			APIKey:    viper.GetString("providers.anthropic.key"),
// 			Model:     string(SupportedModels[model].APIModel),
// 			MaxTokens: maxTokens,
// 		})
//
// 	case ProviderGroq:
// 		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
// 			BaseURL:   "https://api.groq.com/openai/v1",
// 			APIKey:    viper.GetString("providers.groq.key"),
// 			Model:     string(SupportedModels[model].APIModel),
// 			MaxTokens: &maxTokens,
// 		})
//
// 	}
// 	return nil, errors.New("unsupported provider")
// }
