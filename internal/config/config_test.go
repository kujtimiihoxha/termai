package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	setupTest(t)

	t.Run("loads configuration successfully", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		configPath := filepath.Join(homeDir, ".termai.json")

		configContent := `{
			"data": {
				"directory": "custom-dir"
			},
			"log": {
				"level": "debug"
			},
			"mcpServers": {
				"test-server": {
					"command": "test-command",
					"env": ["TEST_ENV=value"],
					"args": ["--arg1", "--arg2"],
					"type": "stdio",
					"url": "",
					"headers": {}
				},
				"sse-server": {
					"command": "",
					"env": [],
					"args": [],
					"type": "sse",
					"url": "https://api.example.com/events",
					"headers": {
						"Authorization": "Bearer token123",
						"Content-Type": "application/json"
					}
				}
			},
			"providers": {
				"anthropic": {
					"apiKey": "test-api-key",
					"enabled": true
				}
			},
			"model": {
				"coder": "claude-3-haiku",
				"task": "claude-3-haiku"
			}
		}`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		cfg = nil
		viper.Reset()

		err = Load(false)
		require.NoError(t, err)

		config := Get()
		assert.NotNil(t, config)
		assert.Equal(t, "custom-dir", config.Data.Directory)
		assert.Equal(t, "debug", config.Log.Level)

		assert.Contains(t, config.MCPServers, "test-server")
		stdioServer := config.MCPServers["test-server"]
		assert.Equal(t, "test-command", stdioServer.Command)
		assert.Equal(t, []string{"TEST_ENV=value"}, stdioServer.Env)
		assert.Equal(t, []string{"--arg1", "--arg2"}, stdioServer.Args)
		assert.Equal(t, MCPStdio, stdioServer.Type)
		assert.Equal(t, "", stdioServer.URL)
		assert.Empty(t, stdioServer.Headers)

		assert.Contains(t, config.MCPServers, "sse-server")
		sseServer := config.MCPServers["sse-server"]
		assert.Equal(t, "", sseServer.Command)
		assert.Empty(t, sseServer.Env)
		assert.Empty(t, sseServer.Args)
		assert.Equal(t, MCPSse, sseServer.Type)
		assert.Equal(t, "https://api.example.com/events", sseServer.URL)
		assert.Equal(t, map[string]string{
			"authorization": "Bearer token123",
			"content-type":  "application/json",
		}, sseServer.Headers)

		assert.Contains(t, config.Providers, models.ModelProvider("anthropic"))
		provider := config.Providers[models.ModelProvider("anthropic")]
		assert.Equal(t, "test-api-key", provider.APIKey)
		assert.True(t, provider.Enabled)

		assert.NotNil(t, config.Model)
		assert.Equal(t, models.Claude3Haiku, config.Model.Coder)
		assert.Equal(t, models.Claude3Haiku, config.Model.Task)
		assert.Equal(t, defaultMaxTokens, config.Model.CoderMaxTokens)
	})

	t.Run("loads configuration with environment variables", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		configPath := filepath.Join(homeDir, ".termai.json")
		err := os.WriteFile(configPath, []byte("{}"), 0o644)
		require.NoError(t, err)

		t.Setenv("ANTHROPIC_API_KEY", "env-anthropic-key")
		t.Setenv("OPENAI_API_KEY", "env-openai-key")
		t.Setenv("GEMINI_API_KEY", "env-gemini-key")

		cfg = nil
		viper.Reset()

		err = Load(false)
		require.NoError(t, err)

		config := Get()
		assert.NotNil(t, config)

		assert.Equal(t, defaultDataDirectory, config.Data.Directory)
		assert.Equal(t, defaultLogLevel, config.Log.Level)

		assert.Contains(t, config.Providers, models.ModelProvider("anthropic"))
		assert.Equal(t, "env-anthropic-key", config.Providers[models.ModelProvider("anthropic")].APIKey)
		assert.True(t, config.Providers[models.ModelProvider("anthropic")].Enabled)

		assert.Contains(t, config.Providers, models.ModelProvider("openai"))
		assert.Equal(t, "env-openai-key", config.Providers[models.ModelProvider("openai")].APIKey)
		assert.True(t, config.Providers[models.ModelProvider("openai")].Enabled)

		assert.Contains(t, config.Providers, models.ModelProvider("gemini"))
		assert.Equal(t, "env-gemini-key", config.Providers[models.ModelProvider("gemini")].APIKey)
		assert.True(t, config.Providers[models.ModelProvider("gemini")].Enabled)

		assert.Equal(t, models.Claude37Sonnet, config.Model.Coder)
	})

	t.Run("local config overrides global config", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		globalConfigPath := filepath.Join(homeDir, ".termai.json")
		globalConfig := `{
			"data": {
				"directory": "global-dir"
			},
			"log": {
				"level": "info"
			}
		}`
		err := os.WriteFile(globalConfigPath, []byte(globalConfig), 0o644)
		require.NoError(t, err)

		workDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(origDir)
		err = os.Chdir(workDir)
		require.NoError(t, err)

		localConfigPath := filepath.Join(workDir, ".termai.json")
		localConfig := `{
			"data": {
				"directory": "local-dir"
			},
			"log": {
				"level": "debug"
			}
		}`
		err = os.WriteFile(localConfigPath, []byte(localConfig), 0o644)
		require.NoError(t, err)

		cfg = nil
		viper.Reset()

		err = Load(false)
		require.NoError(t, err)

		config := Get()
		assert.NotNil(t, config)

		assert.Equal(t, "local-dir", config.Data.Directory)
		assert.Equal(t, "debug", config.Log.Level)
	})

	t.Run("missing config file should not return error", func(t *testing.T) {
		emptyDir := t.TempDir()
		t.Setenv("HOME", emptyDir)

		cfg = nil
		viper.Reset()

		err := Load(false)
		assert.NoError(t, err)
	})

	t.Run("model priority and fallbacks", func(t *testing.T) {
		testCases := []struct {
			name             string
			anthropicKey     string
			openaiKey        string
			geminiKey        string
			expectedModel    models.ModelID
			explicitModel    models.ModelID
			useExplicitModel bool
		}{
			{
				name:          "anthropic has priority",
				anthropicKey:  "test-key",
				openaiKey:     "test-key",
				geminiKey:     "test-key",
				expectedModel: models.Claude37Sonnet,
			},
			{
				name:          "fallback to openai when no anthropic",
				anthropicKey:  "",
				openaiKey:     "test-key",
				geminiKey:     "test-key",
				expectedModel: models.GPT41,
			},
			{
				name:          "fallback to gemini when no others",
				anthropicKey:  "",
				openaiKey:     "",
				geminiKey:     "test-key",
				expectedModel: models.GRMINI20Flash,
			},
			{
				name:             "explicit model overrides defaults",
				anthropicKey:     "test-key",
				openaiKey:        "test-key",
				geminiKey:        "test-key",
				explicitModel:    models.GPT41,
				useExplicitModel: true,
				expectedModel:    models.GPT41,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				homeDir := t.TempDir()
				t.Setenv("HOME", homeDir)
				configPath := filepath.Join(homeDir, ".termai.json")

				configContent := "{}"
				if tc.useExplicitModel {
					configContent = fmt.Sprintf(`{"model":{"coder":"%s"}}`, tc.explicitModel)
				}

				err := os.WriteFile(configPath, []byte(configContent), 0o644)
				require.NoError(t, err)

				if tc.anthropicKey != "" {
					t.Setenv("ANTHROPIC_API_KEY", tc.anthropicKey)
				} else {
					t.Setenv("ANTHROPIC_API_KEY", "")
				}

				if tc.openaiKey != "" {
					t.Setenv("OPENAI_API_KEY", tc.openaiKey)
				} else {
					t.Setenv("OPENAI_API_KEY", "")
				}

				if tc.geminiKey != "" {
					t.Setenv("GEMINI_API_KEY", tc.geminiKey)
				} else {
					t.Setenv("GEMINI_API_KEY", "")
				}

				cfg = nil
				viper.Reset()

				err = Load(false)
				require.NoError(t, err)

				config := Get()
				assert.NotNil(t, config)
				assert.Equal(t, tc.expectedModel, config.Model.Coder)
			})
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("get returns same config instance", func(t *testing.T) {
		setupTest(t)
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		configPath := filepath.Join(homeDir, ".termai.json")
		err := os.WriteFile(configPath, []byte("{}"), 0o644)
		require.NoError(t, err)

		cfg = nil
		viper.Reset()

		config1 := Get()
		require.NotNil(t, config1)

		config2 := Get()
		require.NotNil(t, config2)

		assert.Same(t, config1, config2)
	})

	t.Run("get loads config if not loaded", func(t *testing.T) {
		setupTest(t)
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		configPath := filepath.Join(homeDir, ".termai.json")
		configContent := `{"data":{"directory":"test-dir"}}`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		cfg = nil
		viper.Reset()

		config := Get()
		require.NotNil(t, config)
		assert.Equal(t, "test-dir", config.Data.Directory)
	})
}

func TestWorkingDirectory(t *testing.T) {
	t.Run("returns current working directory", func(t *testing.T) {
		setupTest(t)
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		configPath := filepath.Join(homeDir, ".termai.json")
		err := os.WriteFile(configPath, []byte("{}"), 0o644)
		require.NoError(t, err)

		cfg = nil
		viper.Reset()

		err = Load(false)
		require.NoError(t, err)

		wd := WorkingDirectory()
		expectedWd, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, expectedWd, wd)
	})
}

func TestWrite(t *testing.T) {
	t.Run("writes config to file", func(t *testing.T) {
		setupTest(t)
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		configPath := filepath.Join(homeDir, ".termai.json")
		err := os.WriteFile(configPath, []byte("{}"), 0o644)
		require.NoError(t, err)

		cfg = nil
		viper.Reset()

		err = Load(false)
		require.NoError(t, err)

		viper.Set("data.directory", "modified-dir")

		err = Write()
		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "modified-dir")
	})
}

func TestMCPType(t *testing.T) {
	t.Run("MCPType constants", func(t *testing.T) {
		assert.Equal(t, MCPType("stdio"), MCPStdio)
		assert.Equal(t, MCPType("sse"), MCPSse)
	})

	t.Run("MCPType JSON unmarshaling", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		configPath := filepath.Join(homeDir, ".termai.json")

		configContent := `{
			"mcpServers": {
				"stdio-server": {
					"type": "stdio"
				},
				"sse-server": {
					"type": "sse"
				},
				"invalid-server": {
					"type": "invalid"
				}
			}
		}`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		cfg = nil
		viper.Reset()

		err = Load(false)
		require.NoError(t, err)

		config := Get()
		assert.NotNil(t, config)

		assert.Equal(t, MCPStdio, config.MCPServers["stdio-server"].Type)
		assert.Equal(t, MCPSse, config.MCPServers["sse-server"].Type)
		assert.Equal(t, MCPType("invalid"), config.MCPServers["invalid-server"].Type)
	})

	t.Run("default MCPType", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		configPath := filepath.Join(homeDir, ".termai.json")

		configContent := `{
			"mcpServers": {
				"test-server": {
					"command": "test-command"
				}
			}
		}`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		cfg = nil
		viper.Reset()

		err = Load(false)
		require.NoError(t, err)

		config := Get()
		assert.NotNil(t, config)

		assert.Equal(t, MCPType(""), config.MCPServers["test-server"].Type)
	})
}

func setupTest(t *testing.T) {
	origHome := os.Getenv("HOME")
	origXdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	origAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	origOpenAIKey := os.Getenv("OPENAI_API_KEY")
	origGeminiKey := os.Getenv("GEMINI_API_KEY")

	t.Cleanup(func() {
		t.Setenv("HOME", origHome)
		t.Setenv("XDG_CONFIG_HOME", origXdgConfigHome)
		t.Setenv("ANTHROPIC_API_KEY", origAnthropicKey)
		t.Setenv("OPENAI_API_KEY", origOpenAIKey)
		t.Setenv("GEMINI_API_KEY", origGeminiKey)

		cfg = nil
		viper.Reset()
	})
}
