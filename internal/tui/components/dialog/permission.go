package dialog

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/permission"
	"github.com/kujtimiihoxha/termai/internal/tui/components/core"
	"github.com/kujtimiihoxha/termai/internal/tui/layout"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
	"github.com/kujtimiihoxha/termai/internal/tui/util"

	"github.com/charmbracelet/huh"
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

// PermissionDialog interface for permission dialog component
type PermissionDialog interface {
	tea.Model
	layout.Sizeable
	layout.Bindings
}

// permissionDialogCmp is the implementation of PermissionDialog
type permissionDialogCmp struct {
	form       *huh.Form
	width      int
	height     int
	permission permission.PermissionRequest
	windowSize tea.WindowSizeMsg
	r          *glamour.TermRenderer
}

func (p *permissionDialogCmp) Init() tea.Cmd {
	return nil
}

func (p *permissionDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.windowSize = msg
	default:
		form, cmd := p.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			p.form = f
			cmds = append(cmds, cmd)
		}

		if p.form.State == huh.StateCompleted {
			// Get the selected action
			action := p.form.GetString("action")

			// Close the dialog and return the response
			return p, tea.Batch(
				util.CmdHandler(core.DialogCloseMsg{}),
				util.CmdHandler(PermissionResponseMsg{Action: PermissionAction(action), Permission: p.permission}),
			)
		}
	}

	return p, tea.Batch(cmds...)
}

func (p *permissionDialogCmp) render() string {
	form := p.form.View()
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Rosewater)
	valueStyle := lipgloss.NewStyle().Foreground(styles.Peach)

	headerParts := []string{
		lipgloss.JoinHorizontal(lipgloss.Left, keyStyle.Render("Tool:"), " ", valueStyle.Render(p.permission.ToolName)),
		" ",
		lipgloss.JoinHorizontal(lipgloss.Left, keyStyle.Render("Path:"), " ", valueStyle.Render(p.permission.Path)),
		" ",
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(styles.CatppuccinMarkdownStyle()),
		glamour.WithWordWrap(p.width-10),
		glamour.WithEmoji(),
	)
	content := ""
	switch p.permission.ToolName {
	case tools.BashToolName:
		pr := p.permission.Params.(tools.BashPermissionsParams)
		headerParts = append(headerParts, keyStyle.Render("Command:"))
		content, _ = r.Render(fmt.Sprintf("```bash\n%s\n```", pr.Command))
	case tools.EditToolName:
		pr := p.permission.Params.(tools.EditPermissionsParams)
		headerParts = append(headerParts, keyStyle.Render("Update:"))
		content, _ = r.Render(fmt.Sprintf("```diff\n%s\n```", pr.Diff))
	case tools.WriteToolName:
		pr := p.permission.Params.(tools.WritePermissionsParams)
		headerParts = append(headerParts, keyStyle.Render("Content:"))
		content, _ = r.Render(fmt.Sprintf("```diff\n%s\n```", pr.Content))
	}
	return lipgloss.JoinVertical(
		lipgloss.Top,
		lipgloss.NewStyle().Padding(0, 1).Render(lipgloss.JoinVertical(lipgloss.Left, headerParts...)),
		content,
		form,
	)
}

func (p *permissionDialogCmp) View() string {
	return p.render()
}

func (p *permissionDialogCmp) GetSize() (int, int) {
	return p.width, p.height
}

func (p *permissionDialogCmp) SetSize(width int, height int) {
	p.width = width
	p.height = height
	p.form = p.form.WithWidth(width)
}

func (p *permissionDialogCmp) BindingKeys() []key.Binding {
	return p.form.KeyBinds()
}

func newPermissionDialogCmp(permission permission.PermissionRequest) PermissionDialog {
	// Create a note field for displaying the content

	// Create select field for the permission options
	selectOption := huh.NewSelect[string]().
		Key("action").
		Options(
			huh.NewOption("Allow", string(PermissionAllow)),
			huh.NewOption("Allow for this session", string(PermissionAllowForSession)),
			huh.NewOption("Deny", string(PermissionDeny)),
		).
		Title("Select an action")

	// Apply theme
	theme := styles.HuhTheme()

	// Setup form width and height
	form := huh.NewForm(huh.NewGroup(selectOption)).
		WithShowHelp(false).
		WithTheme(theme).
		WithShowErrors(false)

	// Focus the form for immediate interaction
	selectOption.Focus()

	return &permissionDialogCmp{
		permission: permission,
		form:       form,
	}
}

// NewPermissionDialogCmd creates a new permission dialog command
func NewPermissionDialogCmd(permission permission.PermissionRequest) tea.Cmd {
	permDialog := newPermissionDialogCmp(permission)

	// Create the dialog layout
	dialogPane := layout.NewSinglePane(
		permDialog.(*permissionDialogCmp),
		layout.WithSinglePaneBordered(true),
		layout.WithSinglePaneFocusable(true),
		layout.WithSinglePaneActiveColor(styles.Warning),
		layout.WithSignlePaneBorderText(map[layout.BorderPosition]string{
			layout.TopMiddleBorder: " Permission Required ",
		}),
	)

	// Focus the dialog
	dialogPane.Focus()
	widthRatio := 0.7
	heightRatio := 0.6
	minWidth := 100
	minHeight := 30

	switch permission.ToolName {
	case tools.BashToolName:
		widthRatio = 0.5
		heightRatio = 0.3
		minWidth = 80
		minHeight = 20
	}
	// Return the dialog command
	return util.CmdHandler(core.DialogMsg{
		Content:     dialogPane,
		WidthRatio:  widthRatio,
		HeightRatio: heightRatio,
		MinWidth:    minWidth,
		MinHeight:   minHeight,
	})
}
