package tui

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/permission"
	"github.com/kujtimiihoxha/termai/internal/pubsub"
	"github.com/kujtimiihoxha/termai/internal/tui/components/core"
	"github.com/kujtimiihoxha/termai/internal/tui/components/dialog"
	"github.com/kujtimiihoxha/termai/internal/tui/layout"
	"github.com/kujtimiihoxha/termai/internal/tui/page"
	"github.com/kujtimiihoxha/termai/internal/tui/util"
	"github.com/kujtimiihoxha/vimtea"
)

type keyMap struct {
	Logs   key.Binding
	Return key.Binding
	Back   key.Binding
	Quit   key.Binding
	Help   key.Binding
}

var keys = keyMap{
	Logs: key.NewBinding(
		key.WithKeys("L"),
		key.WithHelp("L", "logs"),
	),
	Return: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close"),
	),
	Back: key.NewBinding(
		key.WithKeys("backspace"),
		key.WithHelp("backspace", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

var editorKeyMap = key.NewBinding(
	key.WithKeys("i"),
	key.WithHelp("i", "insert mode"),
)

type appModel struct {
	width, height int
	currentPage   page.PageID
	previousPage  page.PageID
	pages         map[page.PageID]tea.Model
	loadedPages   map[page.PageID]bool
	status        tea.Model
	help          core.HelpCmp
	dialog        core.DialogCmp
	dialogVisible bool
	editorMode    vimtea.EditorMode
	showHelp      bool
}

func (a appModel) Init() tea.Cmd {
	cmd := a.pages[a.currentPage].Init()
	a.loadedPages[a.currentPage] = true
	return cmd
}

func (a appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pubsub.Event[permission.PermissionRequest]:
		return a, dialog.NewPermissionDialogCmd(msg.Payload)
	case dialog.PermissionResponseMsg:
		switch msg.Action {
		case dialog.PermissionAllow:
			permission.Default.Grant(msg.Permission)
		case dialog.PermissionAllowForSession:
			permission.Default.GrantPersistant(msg.Permission)
		case dialog.PermissionDeny:
			permission.Default.Deny(msg.Permission)
		}
	case vimtea.EditorModeMsg:
		a.editorMode = msg.Mode
	case tea.WindowSizeMsg:
		var cmds []tea.Cmd
		msg.Height -= 1 // Make space for the status bar
		a.width, a.height = msg.Width, msg.Height

		a.status, _ = a.status.Update(msg)

		uh, _ := a.help.Update(msg)
		a.help = uh.(core.HelpCmp)

		p, cmd := a.pages[a.currentPage].Update(msg)
		cmds = append(cmds, cmd)
		a.pages[a.currentPage] = p

		d, cmd := a.dialog.Update(msg)
		cmds = append(cmds, cmd)
		a.dialog = d.(core.DialogCmp)

		return a, tea.Batch(cmds...)
	case core.DialogMsg:
		d, cmd := a.dialog.Update(msg)
		a.dialog = d.(core.DialogCmp)
		a.dialogVisible = true
		return a, cmd
	case core.DialogCloseMsg:
		d, cmd := a.dialog.Update(msg)
		a.dialog = d.(core.DialogCmp)
		a.dialogVisible = false
		return a, cmd
	case page.PageChangeMsg:
		return a, a.moveToPage(msg.ID)
	case util.InfoMsg:
		a.status, _ = a.status.Update(msg)
	case util.ErrorMsg:
		a.status, _ = a.status.Update(msg)
	case tea.KeyMsg:
		if a.editorMode == vimtea.ModeNormal {
			switch {
			case key.Matches(msg, keys.Quit):
				return a, dialog.NewQuitDialogCmd()
			case key.Matches(msg, keys.Back):
				if a.previousPage != "" {
					return a, a.moveToPage(a.previousPage)
				}
			case key.Matches(msg, keys.Return):
				if a.showHelp {
					a.ToggleHelp()
					return a, nil
				}
			case key.Matches(msg, keys.Logs):
				return a, a.moveToPage(page.LogsPage)
			case key.Matches(msg, keys.Help):
				a.ToggleHelp()
				return a, nil
			}
		}
	}
	if a.dialogVisible {
		d, cmd := a.dialog.Update(msg)
		a.dialog = d.(core.DialogCmp)
		return a, cmd
	}
	p, cmd := a.pages[a.currentPage].Update(msg)
	a.pages[a.currentPage] = p
	return a, cmd
}

func (a *appModel) ToggleHelp() {
	if a.showHelp {
		a.showHelp = false
		a.height += a.help.Height()
	} else {
		a.showHelp = true
		a.height -= a.help.Height()
	}

	if sizable, ok := a.pages[a.currentPage].(layout.Sizeable); ok {
		sizable.SetSize(a.width, a.height)
	}
}

func (a *appModel) moveToPage(pageID page.PageID) tea.Cmd {
	var cmd tea.Cmd
	if _, ok := a.loadedPages[pageID]; !ok {
		cmd = a.pages[pageID].Init()
		a.loadedPages[pageID] = true
	}
	a.previousPage = a.currentPage
	a.currentPage = pageID
	if sizable, ok := a.pages[a.currentPage].(layout.Sizeable); ok {
		sizable.SetSize(a.width, a.height)
	}

	return cmd
}

func (a appModel) View() string {
	components := []string{
		a.pages[a.currentPage].View(),
	}

	if a.showHelp {
		bindings := layout.KeyMapToSlice(keys)
		if p, ok := a.pages[a.currentPage].(layout.Bindings); ok {
			bindings = append(bindings, p.BindingKeys()...)
		}
		if a.dialogVisible {
			bindings = append(bindings, a.dialog.BindingKeys()...)
		}
		a.help.SetBindings(bindings)
		components = append(components, a.help.View())
	}

	components = append(components, a.status.View())

	appView := lipgloss.JoinVertical(lipgloss.Top, components...)

	if a.dialogVisible {
		overlay := a.dialog.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}
	return appView
}

func New(app *app.App) tea.Model {
	// Check if config file exists, if not, start with init page
	homedir, _ := os.UserHomeDir()
	configPath := filepath.Join(homedir, ".termai.yaml")

	startPage := page.ReplPage
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		startPage = page.InitPage
	}

	return &appModel{
		currentPage: startPage,
		loadedPages: make(map[page.PageID]bool),
		status:      core.NewStatusCmp(),
		help:        core.NewHelpCmp(),
		dialog:      core.NewDialogCmp(),
		pages: map[page.PageID]tea.Model{
			page.LogsPage: page.NewLogsPage(),
			page.InitPage: page.NewInitPage(),
			page.ReplPage: page.NewReplPage(app),
		},
	}
}
