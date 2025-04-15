package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/spf13/viper"
)

type MCPType string

const (
	MCPStdio MCPType = "stdio"
	MCPSse   MCPType = "sse"
)

type MCPServer struct {
	Command string            `json:"command"`
	Env     []string          `json:"env"`
	Args    []string          `json:"args"`
	Type    MCPType           `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	// TODO: add permissions configuration
	// TODO: add the ability to specify the tools to import
}

type Model struct {
	Coder          models.ModelID `json:"coder"`
	CoderMaxTokens int64          `json:"coderMaxTokens"`

	Task          models.ModelID `json:"task"`
	TaskMaxTokens int64          `json:"taskMaxTokens"`
	// TODO: Maybe support multiple models for different purposes
}

type AnthropicConfig struct {
	DisableCache bool `json:"disableCache"`
	UseBedrock   bool `json:"useBedrock"`
}

type Provider struct {
	APIKey  string `json:"apiKey"`
	Enabled bool   `json:"enabled"`
}

type Data struct {
	Directory string `json:"directory"`
}

type Log struct {
	Level string `json:"level"`
}

type LSPConfig struct {
	Disabled bool     `json:"enabled"`
	Command  string   `json:"command"`
	Args     []string `json:"args"`
	Options  any      `json:"options"`
}

type Config struct {
	Data       *Data                             `json:"data,omitempty"`
	Log        *Log                              `json:"log,omitempty"`
	MCPServers map[string]MCPServer              `json:"mcpServers,omitempty"`
	Providers  map[models.ModelProvider]Provider `json:"providers,omitempty"`

	LSP map[string]LSPConfig `json:"lsp,omitempty"`

	Model *Model `json:"model,omitempty"`

	Debug bool `json:"debug,omitempty"`
}

var cfg *Config

const (
	defaultDataDirectory = ".termai"
	defaultLogLevel      = "info"
	defaultMaxTokens     = int64(5000)
	termai               = "termai"
)

func Load(debug bool) error {
	if cfg != nil {
		return nil
	}

	viper.SetConfigName(fmt.Sprintf(".%s", termai))
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(fmt.Sprintf("$XDG_CONFIG_HOME/%s", termai))
	viper.SetEnvPrefix(strings.ToUpper(termai))

	// Add defaults
	viper.SetDefault("data.directory", defaultDataDirectory)
	if debug {
		viper.SetDefault("debug", true)
		viper.Set("log.level", "debug")
	} else {
		viper.SetDefault("debug", false)
		viper.SetDefault("log.level", defaultLogLevel)
	}

	defaultModelSet := false
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		viper.SetDefault("providers.anthropic.apiKey", os.Getenv("ANTHROPIC_API_KEY"))
		viper.SetDefault("providers.anthropic.enabled", true)
		viper.SetDefault("model.coder", models.Claude37Sonnet)
		viper.SetDefault("model.task", models.Claude37Sonnet)
		defaultModelSet = true
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		viper.SetDefault("providers.openai.apiKey", os.Getenv("OPENAI_API_KEY"))
		viper.SetDefault("providers.openai.enabled", true)
		if !defaultModelSet {
			viper.SetDefault("model.coder", models.GPT41)
			viper.SetDefault("model.task", models.GPT41)
			defaultModelSet = true
		}
	}
	if os.Getenv("GEMINI_API_KEY") != "" {
		viper.SetDefault("providers.gemini.apiKey", os.Getenv("GEMINI_API_KEY"))
		viper.SetDefault("providers.gemini.enabled", true)
		if !defaultModelSet {
			viper.SetDefault("model.coder", models.GRMINI20Flash)
			viper.SetDefault("model.task", models.GRMINI20Flash)
			defaultModelSet = true
		}
	}
	if os.Getenv("GROQ_API_KEY") != "" {
		viper.SetDefault("providers.groq.apiKey", os.Getenv("GROQ_API_KEY"))
		viper.SetDefault("providers.groq.enabled", true)
		if !defaultModelSet {
			viper.SetDefault("model.coder", models.QWENQwq)
			viper.SetDefault("model.task", models.QWENQwq)
			defaultModelSet = true
		}
	}

	viper.SetDefault("providers.bedrock.enabled", true)
	// TODO: add more providers
	cfg = &Config{}

	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	local := viper.New()
	local.SetConfigName(fmt.Sprintf(".%s", termai))
	local.SetConfigType("json")
	local.AddConfigPath(".")
	// load local config, this will override the global config
	if err = local.ReadInConfig(); err == nil {
		viper.MergeConfigMap(local.AllSettings())
	}
	viper.Unmarshal(cfg)

	if cfg.Model != nil && cfg.Model.CoderMaxTokens <= 0 {
		cfg.Model.CoderMaxTokens = defaultMaxTokens
	}
	if cfg.Model != nil && cfg.Model.TaskMaxTokens <= 0 {
		cfg.Model.TaskMaxTokens = defaultMaxTokens
	}

	for _, v := range cfg.MCPServers {
		if v.Type == "" {
			v.Type = MCPStdio
		}
	}

	workdir, err := os.Getwd()
	if err != nil {
		return err
	}
	viper.Set("wd", workdir)
	return nil
}

func Get() *Config {
	if cfg == nil {
		err := Load(false)
		if err != nil {
			panic(err)
		}
	}
	return cfg
}

func WorkingDirectory() string {
	return viper.GetString("wd")
}

func Write() error {
	return viper.WriteConfig()
}
