package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/logging"
	"github.com/kujtimiihoxha/termai/internal/permission"
	"github.com/kujtimiihoxha/termai/internal/pubsub"
	"github.com/kujtimiihoxha/termai/internal/tui/components/core"
	"github.com/kujtimiihoxha/termai/internal/tui/components/dialog"
	"github.com/kujtimiihoxha/termai/internal/tui/components/repl"
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

var replKeyMap = key.NewBinding(
	key.WithKeys("N"),
	key.WithHelp("N", "new session"),
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
	app           *app.App
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
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
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

	// Status
	case util.InfoMsg:
		a.status, cmd = a.status.Update(msg)
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)
	case pubsub.Event[logging.LogMessage]:
		if msg.Payload.Persist {
			switch msg.Payload.Level {
			case "error":
				a.status, cmd = a.status.Update(util.InfoMsg{
					Type: util.InfoTypeError,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})
			case "info":
				a.status, cmd = a.status.Update(util.InfoMsg{
					Type: util.InfoTypeInfo,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})
			case "warn":
				a.status, cmd = a.status.Update(util.InfoMsg{
					Type: util.InfoTypeWarn,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})

			default:
				a.status, cmd = a.status.Update(util.InfoMsg{
					Type: util.InfoTypeInfo,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})
			}
			cmds = append(cmds, cmd)
		}
	case util.ClearStatusMsg:
		a.status, _ = a.status.Update(msg)

	// Permission
	case pubsub.Event[permission.PermissionRequest]:
		return a, dialog.NewPermissionDialogCmd(msg.Payload)
	case dialog.PermissionResponseMsg:
		switch msg.Action {
		case dialog.PermissionAllow:
			a.app.Permissions.Grant(msg.Permission)
		case dialog.PermissionAllowForSession:
			a.app.Permissions.GrantPersistant(msg.Permission)
		case dialog.PermissionDeny:
			a.app.Permissions.Deny(msg.Permission)
		}

	// Dialog
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

	// Editor
	case vimtea.EditorModeMsg:
		a.editorMode = msg.Mode

	case page.PageChangeMsg:
		return a, a.moveToPage(msg.ID)
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
			case key.Matches(msg, replKeyMap):
				if a.currentPage == page.ReplPage {
					sessions, err := a.app.Sessions.List()
					if err != nil {
						return a, util.CmdHandler(util.ReportError(err))
					}
					lastSession := sessions[0]
					if lastSession.MessageCount == 0 {
						return a, util.CmdHandler(repl.SelectedSessionMsg{SessionID: lastSession.ID})
					}
					s, err := a.app.Sessions.Create("New Session")
					if err != nil {
						return a, util.CmdHandler(util.ReportError(err))
					}
					return a, util.CmdHandler(repl.SelectedSessionMsg{SessionID: s.ID})
				}
			// case key.Matches(msg, keys.Logs):
			// 	return a, a.moveToPage(page.LogsPage)
			case msg.String() == "O":
				return a, a.moveToPage(page.ReplPage)
			case key.Matches(msg, keys.Help):
				a.ToggleHelp()
				return a, nil
			}
		}
	}

	if a.dialogVisible {
		d, cmd := a.dialog.Update(msg)
		a.dialog = d.(core.DialogCmp)
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)
	}
	a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(msg)
	cmds = append(cmds, cmd)
	return a, tea.Batch(cmds...)
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
		if a.currentPage == page.ReplPage {
			bindings = append(bindings, replKeyMap)
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
	// homedir, _ := os.UserHomeDir()
	// configPath := filepath.Join(homedir, ".termai.yaml")
	//
	startPage := page.ChatPage
	// if _, err := os.Stat(configPath); os.IsNotExist(err) {
	// 	startPage = page.InitPage
	// }

	return &appModel{
		currentPage: startPage,
		loadedPages: make(map[page.PageID]bool),
		status:      core.NewStatusCmp(),
		help:        core.NewHelpCmp(),
		dialog:      core.NewDialogCmp(),
		app:         app,
		pages: map[page.PageID]tea.Model{
			page.ChatPage: page.NewChatPage(app),
			page.LogsPage: page.NewLogsPage(),
			page.InitPage: page.NewInitPage(),
			page.ReplPage: page.NewReplPage(app),
		},
	}
}
