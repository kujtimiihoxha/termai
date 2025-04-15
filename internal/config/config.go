// Package config manages application configuration from various sources.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/logging"
	"github.com/spf13/viper"
)

// MCPType defines the type of MCP (Model Control Protocol) server.
type MCPType string

// Supported MCP types
const (
	MCPStdio MCPType = "stdio"
	MCPSse   MCPType = "sse"
)

// MCPServer defines the configuration for a Model Control Protocol server.
type MCPServer struct {
	Command string            `json:"command"`
	Env     []string          `json:"env"`
	Args    []string          `json:"args"`
	Type    MCPType           `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// Model defines configuration for different LLM models and their token limits.
type Model struct {
	Coder          models.ModelID `json:"coder"`
	CoderMaxTokens int64          `json:"coderMaxTokens"`
	Task           models.ModelID `json:"task"`
	TaskMaxTokens  int64          `json:"taskMaxTokens"`
}

// Provider defines configuration for an LLM provider.
type Provider struct {
	APIKey   string `json:"apiKey"`
	Disabled bool   `json:"disabled"`
}

// Data defines storage configuration.
type Data struct {
	Directory string `json:"directory"`
}

// LSPConfig defines configuration for Language Server Protocol integration.
type LSPConfig struct {
	Disabled bool     `json:"enabled"`
	Command  string   `json:"command"`
	Args     []string `json:"args"`
	Options  any      `json:"options"`
}

// Config is the main configuration structure for the application.
type Config struct {
	Data       Data                              `json:"data"`
	WorkingDir string                            `json:"wd,omitempty"`
	MCPServers map[string]MCPServer              `json:"mcpServers,omitempty"`
	Providers  map[models.ModelProvider]Provider `json:"providers,omitempty"`
	LSP        map[string]LSPConfig              `json:"lsp,omitempty"`
	Model      Model                             `json:"model"`
	Debug      bool                              `json:"debug,omitempty"`
}

// Application constants
const (
	defaultDataDirectory = ".opencode"
	defaultLogLevel      = "info"
	defaultMaxTokens     = int64(5000)
	appName              = "opencode"
)

// Global configuration instance
var cfg *Config

// Load initializes the configuration from environment variables and config files.
// If debug is true, debug mode is enabled and log level is set to debug.
// It returns an error if configuration loading fails.
func Load(workingDir string, debug bool) (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		WorkingDir: workingDir,
		MCPServers: make(map[string]MCPServer),
		Providers:  make(map[models.ModelProvider]Provider),
		LSP:        make(map[string]LSPConfig),
	}

	configureViper()
	setDefaults(debug)
	setProviderDefaults()

	// Read global config
	if err := readConfig(viper.ReadInConfig()); err != nil {
		return cfg, err
	}

	// Load and merge local config
	mergeLocalConfig(workingDir)

	// Apply configuration to the struct
	if err := viper.Unmarshal(cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	applyDefaultValues()

	defaultLevel := slog.LevelInfo
	if cfg.Debug {
		defaultLevel = slog.LevelDebug
	}
	// Configure logger
	logger := slog.New(slog.NewTextHandler(logging.NewWriter(), &slog.HandlerOptions{
		Level: defaultLevel,
	}))
	slog.SetDefault(logger)
	return cfg, nil
}

// configureViper sets up viper's configuration paths and environment variables.
func configureViper() {
	viper.SetConfigName(fmt.Sprintf(".%s", appName))
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(fmt.Sprintf("$XDG_CONFIG_HOME/%s", appName))
	viper.SetEnvPrefix(strings.ToUpper(appName))
	viper.AutomaticEnv()
}

// setDefaults configures default values for configuration options.
func setDefaults(debug bool) {
	viper.SetDefault("data.directory", defaultDataDirectory)

	if debug {
		viper.SetDefault("debug", true)
		viper.Set("log.level", "debug")
	} else {
		viper.SetDefault("debug", false)
		viper.SetDefault("log.level", defaultLogLevel)
	}
}

// setProviderDefaults configures LLM provider defaults based on environment variables.
// the default model priority is:
// 1. Anthropic
// 2. OpenAI
// 3. Google Gemini
// 4. AWS Bedrock
func setProviderDefaults() {
	// Groq configuration
	if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		viper.SetDefault("providers.groq.apiKey", apiKey)
		viper.SetDefault("model.coder", models.QWENQwq)
		viper.SetDefault("model.coderMaxTokens", defaultMaxTokens)
		viper.SetDefault("model.task", models.QWENQwq)
		viper.SetDefault("model.taskMaxTokens", defaultMaxTokens)
	}

	// Google Gemini configuration
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		viper.SetDefault("providers.gemini.apiKey", apiKey)
		viper.SetDefault("model.coder", models.GRMINI20Flash)
		viper.SetDefault("model.coderMaxTokens", defaultMaxTokens)
		viper.SetDefault("model.task", models.GRMINI20Flash)
		viper.SetDefault("model.taskMaxTokens", defaultMaxTokens)
	}

	// OpenAI configuration
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		viper.SetDefault("providers.openai.apiKey", apiKey)
		viper.SetDefault("model.coder", models.GPT4o)
		viper.SetDefault("model.coderMaxTokens", defaultMaxTokens)
		viper.SetDefault("model.task", models.GPT4o)
		viper.SetDefault("model.taskMaxTokens", defaultMaxTokens)
	}

	// Anthropic configuration
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		viper.SetDefault("providers.anthropic.apiKey", apiKey)
		viper.SetDefault("model.coder", models.Claude37Sonnet)
		viper.SetDefault("model.coderMaxTokens", defaultMaxTokens)
		viper.SetDefault("model.task", models.Claude37Sonnet)
		viper.SetDefault("model.taskMaxTokens", defaultMaxTokens)
	}

	if hasAWSCredentials() {
		viper.SetDefault("model.coder", models.BedrockClaude37Sonnet)
		viper.SetDefault("model.coderMaxTokens", defaultMaxTokens)
		viper.SetDefault("model.task", models.BedrockClaude37Sonnet)
		viper.SetDefault("model.taskMaxTokens", defaultMaxTokens)
	}
}

// hasAWSCredentials checks if AWS credentials are available in the environment.
func hasAWSCredentials() bool {
	// Check for explicit AWS credentials
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}

	// Check for AWS profile
	if os.Getenv("AWS_PROFILE") != "" || os.Getenv("AWS_DEFAULT_PROFILE") != "" {
		return true
	}

	// Check for AWS region
	if os.Getenv("AWS_REGION") != "" || os.Getenv("AWS_DEFAULT_REGION") != "" {
		return true
	}

	// Check if running on EC2 with instance profile
	if os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") != "" ||
		os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI") != "" {
		return true
	}

	return false
}

// readConfig handles the result of reading a configuration file.
func readConfig(err error) error {
	if err == nil {
		return nil
	}

	// It's okay if the config file doesn't exist
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		return nil
	}

	return fmt.Errorf("failed to read config: %w", err)
}

// mergeLocalConfig loads and merges configuration from the local directory.
func mergeLocalConfig(workingDir string) {
	local := viper.New()
	local.SetConfigName(fmt.Sprintf(".%s", appName))
	local.SetConfigType("json")
	local.AddConfigPath(workingDir)

	// Merge local config if it exists
	if err := local.ReadInConfig(); err == nil {
		viper.MergeConfigMap(local.AllSettings())
	}
}

// applyDefaultValues sets default values for configuration fields that need processing.
func applyDefaultValues() {
	// Set default MCP type if not specified
	for k, v := range cfg.MCPServers {
		if v.Type == "" {
			v.Type = MCPStdio
			cfg.MCPServers[k] = v
		}
	}
}

// Get returns the current configuration.
// It's safe to call this function multiple times.
func Get() *Config {
	return cfg
}

// WorkingDirectory returns the current working directory from the configuration.
func WorkingDirectory() string {
	if cfg == nil {
		panic("config not loaded")
	}
	return cfg.WorkingDir
}
