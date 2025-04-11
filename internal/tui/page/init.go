package page

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/tui/layout"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
	"github.com/kujtimiihoxha/termai/internal/tui/util"
	"github.com/spf13/viper"
)

var InitPage PageID = "init"

type configSaved struct{}

type initPage struct {
	form         *huh.Form
	width        int
	height       int
	saved        bool
	errorMsg     string
	statusMsg    string
	modelOpts    []huh.Option[string]
	bigModel     string
	smallModel   string
	openAIKey    string
	anthropicKey string
	groqKey      string
	maxTokens    string
	dataDir      string
	agent        string
}

func (i *initPage) Init() tea.Cmd {
	return i.form.Init()
}

func (i *initPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		i.width = msg.Width - 4 // Account for border
		i.height = msg.Height - 4
		i.form = i.form.WithWidth(i.width).WithHeight(i.height)
		return i, nil

	case configSaved:
		i.saved = true
		i.statusMsg = "Configuration saved successfully. Press any key to continue."
		return i, nil
	}

	if i.saved {
		switch msg.(type) {
		case tea.KeyMsg:
			return i, util.CmdHandler(PageChangeMsg{ID: ReplPage})
		}
		return i, nil
	}

	// Process the form
	form, cmd := i.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		i.form = f
		cmds = append(cmds, cmd)
	}

	if i.form.State == huh.StateCompleted {
		// Save configuration to file
		configPath := filepath.Join(os.Getenv("HOME"), ".termai.yaml")
		maxTokens, _ := strconv.Atoi(i.maxTokens)
		config := map[string]any{
			"models": map[string]string{
				"big":   i.bigModel,
				"small": i.smallModel,
			},
			"providers": map[string]any{
				"openai": map[string]string{
					"key": i.openAIKey,
				},
				"anthropic": map[string]string{
					"key": i.anthropicKey,
				},
				"groq": map[string]string{
					"key": i.groqKey,
				},
				"common": map[string]int{
					"max_tokens": maxTokens,
				},
			},
			"data": map[string]string{
				"dir": i.dataDir,
			},
			"agents": map[string]string{
				"default": i.agent,
			},
			"log": map[string]string{
				"level": "info",
			},
		}

		// Write config to viper
		for k, v := range config {
			viper.Set(k, v)
		}

		// Save configuration
		err := viper.WriteConfigAs(configPath)
		if err != nil {
			i.errorMsg = fmt.Sprintf("Failed to save configuration: %s", err)
			return i, nil
		}

		// Return to main page
		return i, util.CmdHandler(configSaved{})
	}

	return i, tea.Batch(cmds...)
}

func (i *initPage) View() string {
	if i.saved {
		return lipgloss.NewStyle().
			Width(i.width).
			Height(i.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(lipgloss.JoinVertical(
				lipgloss.Center,
				lipgloss.NewStyle().Foreground(styles.Green).Render("✓ Configuration Saved"),
				"",
				lipgloss.NewStyle().Foreground(styles.Blue).Render(i.statusMsg),
			))
	}

	view := i.form.View()
	if i.errorMsg != "" {
		errorBox := lipgloss.NewStyle().
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Red).
			Width(i.width - 4).
			Render(i.errorMsg)
		view = lipgloss.JoinVertical(lipgloss.Left, errorBox, view)
	}
	return view
}

func (i *initPage) GetSize() (int, int) {
	return i.width, i.height
}

func (i *initPage) SetSize(width int, height int) {
	i.width = width
	i.height = height
	i.form = i.form.WithWidth(width).WithHeight(height)
}

func (i *initPage) BindingKeys() []key.Binding {
	if i.saved {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter", "space", "esc"),
				key.WithHelp("any key", "continue"),
			),
		}
	}
	return i.form.KeyBinds()
}

func NewInitPage() tea.Model {
	// Create model options
	var modelOpts []huh.Option[string]
	for id, model := range models.SupportedModels {
		modelOpts = append(modelOpts, huh.NewOption(model.Name, string(id)))
	}

	// Create agent options
	agentOpts := []huh.Option[string]{
		huh.NewOption("Coder", "coder"),
		huh.NewOption("Assistant", "assistant"),
	}

	// Init page with form
	initModel := &initPage{
		modelOpts:  modelOpts,
		bigModel:   string(models.Claude37Sonnet),
		smallModel: string(models.Claude37Sonnet),
		maxTokens:  "4000",
		dataDir:    ".termai",
		agent:      "coder",
	}

	// API Keys group
	apiKeysGroup := huh.NewGroup(
		huh.NewNote().
			Title("API Keys").
			Description("You need to provide at least one API key to use termai"),

		huh.NewInput().
			Title("OpenAI API Key").
			Placeholder("sk-...").
			Key("openai_key").
			Value(&initModel.openAIKey),

		huh.NewInput().
			Title("Anthropic API Key").
			Placeholder("sk-ant-...").
			Key("anthropic_key").
			Value(&initModel.anthropicKey),

		huh.NewInput().
			Title("Groq API Key").
			Placeholder("gsk_...").
			Key("groq_key").
			Value(&initModel.groqKey),
	)

	// Model configuration group
	modelsGroup := huh.NewGroup(
		huh.NewNote().
			Title("Model Configuration").
			Description("Select which models to use"),

		huh.NewSelect[string]().
			Title("Big Model").
			Options(modelOpts...).
			Key("big_model").
			Value(&initModel.bigModel),

		huh.NewSelect[string]().
			Title("Small Model").
			Options(modelOpts...).
			Key("small_model").
			Value(&initModel.smallModel),

		huh.NewInput().
			Title("Max Tokens").
			Placeholder("4000").
			Key("max_tokens").
			CharLimit(5).
			Validate(func(s string) error {
				var n int
				_, err := fmt.Sscanf(s, "%d", &n)
				if err != nil || n <= 0 {
					return fmt.Errorf("must be a positive number")
				}
				initModel.maxTokens = s
				return nil
			}).
			Value(&initModel.maxTokens),
	)

	// General settings group
	generalGroup := huh.NewGroup(
		huh.NewNote().
			Title("General Settings").
			Description("Configure general termai settings"),

		huh.NewInput().
			Title("Data Directory").
			Placeholder(".termai").
			Key("data_dir").
			Value(&initModel.dataDir),

		huh.NewSelect[string]().
			Title("Default Agent").
			Options(agentOpts...).
			Key("agent").
			Value(&initModel.agent),

		huh.NewConfirm().
			Title("Save Configuration").
			Affirmative("Save").
			Negative("Cancel"),
	)

	// Create form with theme
	form := huh.NewForm(
		apiKeysGroup,
		modelsGroup,
		generalGroup,
	).WithTheme(styles.HuhTheme()).
		WithShowHelp(true).
		WithShowErrors(true)

	// Set the form in the model
	initModel.form = form

	return layout.NewSinglePane(
		initModel,
		layout.WithSinglePaneFocusable(true),
		layout.WithSinglePaneBordered(true),
		layout.WithSinglePaneBorderText(
			map[layout.BorderPosition]string{
				layout.TopMiddleBorder: "Welcome to termai - Initial Setup",
			},
		),
	)
}
