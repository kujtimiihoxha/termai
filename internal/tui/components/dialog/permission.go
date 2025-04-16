package dialog

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/opencode/internal/diff"
	"github.com/kujtimiihoxha/opencode/internal/llm/tools"
	"github.com/kujtimiihoxha/opencode/internal/permission"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
	"github.com/kujtimiihoxha/opencode/internal/tui/util"
)

type PermissionAction string

// Permission responses
const (
	PermissionAllow           PermissionAction = "allow"
	PermissionAllowForSession PermissionAction = "allow_session"
	PermissionDeny            PermissionAction = "deny"
)

// PermissionResponseMsg represents the user's response to a permission request
type PermissionResponseMsg struct {
	Permission permission.PermissionRequest
	Action     PermissionAction
}

// PermissionDialogCmp interface for permission dialog component
type PermissionDialogCmp interface {
	tea.Model
	layout.Bindings
	SetPermissions(permission permission.PermissionRequest)
}

type permissionsMapping struct {
	LeftRight    key.Binding
	EnterSpace   key.Binding
	Allow        key.Binding
	AllowSession key.Binding
	Deny         key.Binding
	Tab          key.Binding
}

var permissionsKeys = permissionsMapping{
	LeftRight: key.NewBinding(
		key.WithKeys("left", "right"),
		key.WithHelp("←/→", "switch options"),
	),
	EnterSpace: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "confirm"),
	),
	Allow: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "allow"),
	),
	AllowSession: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "allow for session"),
	),
	Deny: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "deny"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch options"),
	),
}

// permissionDialogCmp is the implementation of PermissionDialog
type permissionDialogCmp struct {
	width           int
	height          int
	permission      permission.PermissionRequest
	windowSize      tea.WindowSizeMsg
	contentViewPort viewport.Model
	selectedOption  int // 0: Allow, 1: Allow for session, 2: Deny

	diffCache     map[string]string
	markdownCache map[string]string
}

func (p *permissionDialogCmp) Init() tea.Cmd {
	return p.contentViewPort.Init()
}

func (p *permissionDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.windowSize = msg
		p.SetSize()
		p.markdownCache = make(map[string]string)
		p.diffCache = make(map[string]string)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, permissionsKeys.LeftRight) || key.Matches(msg, permissionsKeys.Tab):
			// Change selected option
			p.selectedOption = (p.selectedOption + 1) % 3
			return p, nil
		case key.Matches(msg, permissionsKeys.EnterSpace):
			// Select current option
			return p, p.selectCurrentOption()
		case key.Matches(msg, permissionsKeys.Allow):
			// Select Allow
			return p, util.CmdHandler(PermissionResponseMsg{Action: PermissionAllow, Permission: p.permission})
		case key.Matches(msg, permissionsKeys.AllowSession):
			// Select Allow for session
			return p, util.CmdHandler(PermissionResponseMsg{Action: PermissionAllowForSession, Permission: p.permission})
		case key.Matches(msg, permissionsKeys.Deny):
			// Select Deny
			return p, util.CmdHandler(PermissionResponseMsg{Action: PermissionDeny, Permission: p.permission})
		default:
			// Pass other keys to viewport
			viewPort, cmd := p.contentViewPort.Update(msg)
			p.contentViewPort = viewPort
			cmds = append(cmds, cmd)
		}
	}

	return p, tea.Batch(cmds...)
}

func (p *permissionDialogCmp) selectCurrentOption() tea.Cmd {
	var action PermissionAction

	switch p.selectedOption {
	case 0:
		action = PermissionAllow
	case 1:
		action = PermissionAllowForSession
	case 2:
		action = PermissionDeny
	}

	return util.CmdHandler(PermissionResponseMsg{Action: action, Permission: p.permission})
}

func (p *permissionDialogCmp) renderButtons() string {
	allowStyle := styles.BaseStyle
	allowSessionStyle := styles.BaseStyle
	denyStyle := styles.BaseStyle
	spacerStyle := styles.BaseStyle.Background(styles.Background)

	// Style the selected button
	switch p.selectedOption {
	case 0:
		allowStyle = allowStyle.Background(styles.PrimaryColor).Foreground(styles.Background)
		allowSessionStyle = allowSessionStyle.Background(styles.Background).Foreground(styles.PrimaryColor)
		denyStyle = denyStyle.Background(styles.Background).Foreground(styles.PrimaryColor)
	case 1:
		allowStyle = allowStyle.Background(styles.Background).Foreground(styles.PrimaryColor)
		allowSessionStyle = allowSessionStyle.Background(styles.PrimaryColor).Foreground(styles.Background)
		denyStyle = denyStyle.Background(styles.Background).Foreground(styles.PrimaryColor)
	case 2:
		allowStyle = allowStyle.Background(styles.Background).Foreground(styles.PrimaryColor)
		allowSessionStyle = allowSessionStyle.Background(styles.Background).Foreground(styles.PrimaryColor)
		denyStyle = denyStyle.Background(styles.PrimaryColor).Foreground(styles.Background)
	}

	allowButton := allowStyle.Padding(0, 1).Render("Allow (a)")
	allowSessionButton := allowSessionStyle.Padding(0, 1).Render("Allow for session (A)")
	denyButton := denyStyle.Padding(0, 1).Render("Deny (d)")

	content := lipgloss.JoinHorizontal(
		lipgloss.Left,
		allowButton,
		spacerStyle.Render("  "),
		allowSessionButton,
		spacerStyle.Render("  "),
		denyButton,
		spacerStyle.Render("  "),
	)

	remainingWidth := p.width - lipgloss.Width(content)
	if remainingWidth > 0 {
		content = spacerStyle.Render(strings.Repeat(" ", remainingWidth)) + content
	}
	return content
}

func (p *permissionDialogCmp) renderHeader() string {
	toolKey := styles.BaseStyle.Foreground(styles.ForgroundDim).Bold(true).Render("Tool")
	toolValue := styles.BaseStyle.
		Foreground(styles.Forground).
		Width(p.width - lipgloss.Width(toolKey)).
		Render(fmt.Sprintf(": %s", p.permission.ToolName))

	pathKey := styles.BaseStyle.Foreground(styles.ForgroundDim).Bold(true).Render("Path")
	pathValue := styles.BaseStyle.
		Foreground(styles.Forground).
		Width(p.width - lipgloss.Width(pathKey)).
		Render(fmt.Sprintf(": %s", p.permission.Path))

	headerParts := []string{
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			toolKey,
			toolValue,
		),
		styles.BaseStyle.Render(strings.Repeat(" ", p.width)),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			pathKey,
			pathValue,
		),
		styles.BaseStyle.Render(strings.Repeat(" ", p.width)),
	}

	// Add tool-specific header information
	switch p.permission.ToolName {
	case tools.BashToolName:
		headerParts = append(headerParts, styles.BaseStyle.Foreground(styles.ForgroundDim).Width(p.width).Bold(true).Render("Command"))
	case tools.EditToolName:
		headerParts = append(headerParts, styles.BaseStyle.Foreground(styles.ForgroundDim).Width(p.width).Bold(true).Render("Diff"))
	case tools.WriteToolName:
		headerParts = append(headerParts, styles.BaseStyle.Foreground(styles.ForgroundDim).Width(p.width).Bold(true).Render("Diff"))
	case tools.FetchToolName:
		headerParts = append(headerParts, styles.BaseStyle.Foreground(styles.ForgroundDim).Width(p.width).Bold(true).Render("URL"))
	}

	return lipgloss.NewStyle().Render(lipgloss.JoinVertical(lipgloss.Left, headerParts...))
}

func (p *permissionDialogCmp) renderBashContent() string {
	if pr, ok := p.permission.Params.(tools.BashPermissionsParams); ok {
		content := fmt.Sprintf("```bash\n%s\n```", pr.Command)

		// Use the cache for markdown rendering
		renderedContent := p.GetOrSetMarkdown(p.permission.ID, func() (string, error) {
			r, _ := glamour.NewTermRenderer(
				glamour.WithStyles(styles.MarkdownTheme(true)),
				glamour.WithWordWrap(p.width-10),
			)
			s, err := r.Render(content)
			return styles.ForceReplaceBackgroundWithLipgloss(s, styles.Background), err
		})

		finalContent := styles.BaseStyle.
			Width(p.contentViewPort.Width).
			Render(renderedContent)
		p.contentViewPort.SetContent(finalContent)
		return p.styleViewport()
	}
	return ""
}

func (p *permissionDialogCmp) renderEditContent() string {
	if pr, ok := p.permission.Params.(tools.EditPermissionsParams); ok {
		diff := p.GetOrSetDiff(p.permission.ID, func() (string, error) {
			return diff.FormatDiff(pr.Diff, diff.WithTotalWidth(p.contentViewPort.Width))
		})

		p.contentViewPort.SetContent(diff)
		return p.styleViewport()
	}
	return ""
}

func (p *permissionDialogCmp) renderWriteContent() string {
	if pr, ok := p.permission.Params.(tools.WritePermissionsParams); ok {
		// Use the cache for diff rendering
		diff := p.GetOrSetDiff(p.permission.ID, func() (string, error) {
			return diff.FormatDiff(pr.Diff, diff.WithTotalWidth(p.contentViewPort.Width))
		})

		p.contentViewPort.SetContent(diff)
		return p.styleViewport()
	}
	return ""
}

func (p *permissionDialogCmp) renderFetchContent() string {
	if pr, ok := p.permission.Params.(tools.FetchPermissionsParams); ok {
		content := fmt.Sprintf("```bash\n%s\n```", pr.URL)

		// Use the cache for markdown rendering
		renderedContent := p.GetOrSetMarkdown(p.permission.ID, func() (string, error) {
			r, _ := glamour.NewTermRenderer(
				glamour.WithStyles(styles.MarkdownTheme(true)),
				glamour.WithWordWrap(p.width-10),
			)
			s, err := r.Render(content)
			return styles.ForceReplaceBackgroundWithLipgloss(s, styles.Background), err
		})

		p.contentViewPort.SetContent(renderedContent)
		return p.styleViewport()
	}
	return ""
}

func (p *permissionDialogCmp) renderDefaultContent() string {
	content := p.permission.Description

	// Use the cache for markdown rendering
	renderedContent := p.GetOrSetMarkdown(p.permission.ID, func() (string, error) {
		r, _ := glamour.NewTermRenderer(
			glamour.WithStyles(styles.CatppuccinMarkdownStyle()),
			glamour.WithWordWrap(p.width-10),
		)
		s, err := r.Render(content)
		return styles.ForceReplaceBackgroundWithLipgloss(s, styles.Background), err
	})

	p.contentViewPort.SetContent(renderedContent)

	if renderedContent == "" {
		return ""
	}

	return p.styleViewport()
}

func (p *permissionDialogCmp) styleViewport() string {
	contentStyle := lipgloss.NewStyle().
		Background(styles.Background)

	return contentStyle.Render(p.contentViewPort.View())
}

func (p *permissionDialogCmp) render() string {
	title := styles.BaseStyle.
		Bold(true).
		Width(p.width - 4).
		Foreground(styles.PrimaryColor).
		Render("Permission Required")
	// Render header
	headerContent := p.renderHeader()
	// Render buttons
	buttons := p.renderButtons()

	// Calculate content height dynamically based on window size
	p.contentViewPort.Height = p.height - lipgloss.Height(headerContent) - lipgloss.Height(buttons) - 2 - lipgloss.Height(title)
	p.contentViewPort.Width = p.width - 4

	// Render content based on tool type
	var contentFinal string
	switch p.permission.ToolName {
	case tools.BashToolName:
		contentFinal = p.renderBashContent()
	case tools.EditToolName:
		contentFinal = p.renderEditContent()
	case tools.WriteToolName:
		contentFinal = p.renderWriteContent()
	case tools.FetchToolName:
		contentFinal = p.renderFetchContent()
	default:
		contentFinal = p.renderDefaultContent()
	}

	content := lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		styles.BaseStyle.Render(strings.Repeat(" ", lipgloss.Width(title))),
		headerContent,
		contentFinal,
		buttons,
	)

	return styles.BaseStyle.
		Padding(1, 0, 0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(styles.Background).
		BorderForeground(styles.ForgroundDim).
		Width(p.width).
		Height(p.height).
		Render(
			content,
		)
}

func (p *permissionDialogCmp) View() string {
	return p.render()
}

func (p *permissionDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(helpKeys)
}

func (p *permissionDialogCmp) SetSize() {
	if p.permission.ID == "" {
		return
	}
	switch p.permission.ToolName {
	case tools.BashToolName:
		p.width = int(float64(p.windowSize.Width) * 0.4)
		p.height = int(float64(p.windowSize.Height) * 0.3)
	case tools.EditToolName:
		p.width = int(float64(p.windowSize.Width) * 0.8)
		p.height = int(float64(p.windowSize.Height) * 0.8)
	case tools.WriteToolName:
		p.width = int(float64(p.windowSize.Width) * 0.8)
		p.height = int(float64(p.windowSize.Height) * 0.8)
	case tools.FetchToolName:
		p.width = int(float64(p.windowSize.Width) * 0.4)
		p.height = int(float64(p.windowSize.Height) * 0.3)
	default:
		p.width = int(float64(p.windowSize.Width) * 0.7)
		p.height = int(float64(p.windowSize.Height) * 0.5)
	}
}

func (p *permissionDialogCmp) SetPermissions(permission permission.PermissionRequest) {
	p.permission = permission
	p.SetSize()
}

// Helper to get or set cached diff content
func (c *permissionDialogCmp) GetOrSetDiff(key string, generator func() (string, error)) string {
	if cached, ok := c.diffCache[key]; ok {
		return cached
	}

	content, err := generator()
	if err != nil {
		return fmt.Sprintf("Error formatting diff: %v", err)
	}

	c.diffCache[key] = content

	return content
}

// Helper to get or set cached markdown content
func (c *permissionDialogCmp) GetOrSetMarkdown(key string, generator func() (string, error)) string {
	if cached, ok := c.markdownCache[key]; ok {
		return cached
	}

	content, err := generator()
	if err != nil {
		return fmt.Sprintf("Error rendering markdown: %v", err)
	}

	c.markdownCache[key] = content

	return content
}

func NewPermissionDialogCmp() PermissionDialogCmp {
	// Create viewport for content
	contentViewport := viewport.New(0, 0)

	return &permissionDialogCmp{
		contentViewPort: contentViewport,
		selectedOption:  0, // Default to "Allow"
		diffCache:       make(map[string]string),
		markdownCache:   make(map[string]string),
	}
}
