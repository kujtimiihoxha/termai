package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/opencode/internal/app"
	"github.com/kujtimiihoxha/opencode/internal/logging"
	"github.com/kujtimiihoxha/opencode/internal/permission"
	"github.com/kujtimiihoxha/opencode/internal/pubsub"
	"github.com/kujtimiihoxha/opencode/internal/tui/components/core"
	"github.com/kujtimiihoxha/opencode/internal/tui/components/dialog"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
	"github.com/kujtimiihoxha/opencode/internal/tui/page"
	"github.com/kujtimiihoxha/opencode/internal/tui/util"
)

type keyMap struct {
	Logs key.Binding
	Quit key.Binding
	Help key.Binding
}

var keys = keyMap{
	Logs: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+L", "logs"),
	),

	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("ctrl+_"),
		key.WithHelp("ctrl+?", "toggle help"),
	),
}

var returnKey = key.NewBinding(
	key.WithKeys("esc"),
	key.WithHelp("esc", "close"),
)

var logsKeyReturnKey = key.NewBinding(
	key.WithKeys("backspace"),
	key.WithHelp("backspace", "go back"),
)

type appModel struct {
	width, height int
	currentPage   page.PageID
	previousPage  page.PageID
	pages         map[page.PageID]tea.Model
	loadedPages   map[page.PageID]bool
	status        tea.Model
	app           *app.App

	showPermissions bool
	permissions     dialog.PermissionDialogCmp

	showHelp bool
	help     dialog.HelpCmp

	showQuit bool
	quit     dialog.QuitDialog
}

func (a appModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmd := a.pages[a.currentPage].Init()
	a.loadedPages[a.currentPage] = true
	cmds = append(cmds, cmd)
	cmd = a.status.Init()
	cmds = append(cmds, cmd)
	cmd = a.quit.Init()
	cmds = append(cmds, cmd)
	cmd = a.help.Init()
	cmds = append(cmds, cmd)
	return tea.Batch(cmds...)
}

func (a appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		msg.Height -= 1 // Make space for the status bar
		a.width, a.height = msg.Width, msg.Height

		a.status, _ = a.status.Update(msg)
		a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(msg)
		cmds = append(cmds, cmd)

		prm, permCmd := a.permissions.Update(msg)
		a.permissions = prm.(dialog.PermissionDialogCmp)
		cmds = append(cmds, permCmd)

		help, helpCmd := a.help.Update(msg)
		a.help = help.(dialog.HelpCmp)
		cmds = append(cmds, helpCmd)

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
		a.showPermissions = true
		a.permissions.SetPermissions(msg.Payload)
		return a, nil
	case dialog.PermissionResponseMsg:
		switch msg.Action {
		case dialog.PermissionAllow:
			a.app.Permissions.Grant(msg.Permission)
		case dialog.PermissionAllowForSession:
			a.app.Permissions.GrantPersistant(msg.Permission)
		case dialog.PermissionDeny:
			a.app.Permissions.Deny(msg.Permission)
		}
		a.showPermissions = false
		return a, nil

	case page.PageChangeMsg:
		return a, a.moveToPage(msg.ID)

	case dialog.CloseQuitMsg:
		a.showQuit = false
		return a, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			a.showQuit = !a.showQuit
			if a.showHelp {
				a.showHelp = false
			}
			return a, nil
		case key.Matches(msg, logsKeyReturnKey):
			if a.currentPage == page.LogsPage {
				return a, a.moveToPage(page.ChatPage)
			}
		case key.Matches(msg, returnKey):
			if a.showQuit {
				a.showQuit = !a.showQuit
				return a, nil
			}
			if a.showHelp {
				a.showHelp = !a.showHelp
				return a, nil
			}
		case key.Matches(msg, keys.Logs):
			return a, a.moveToPage(page.LogsPage)
		case key.Matches(msg, keys.Help):
			if a.showQuit {
				return a, nil
			}
			a.showHelp = !a.showHelp
			return a, nil
		}
	}

	if a.showQuit {
		q, quitCmd := a.quit.Update(msg)
		a.quit = q.(dialog.QuitDialog)
		cmds = append(cmds, quitCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}
	if a.showPermissions {
		d, permissionsCmd := a.permissions.Update(msg)
		a.permissions = d.(dialog.PermissionDialogCmp)
		cmds = append(cmds, permissionsCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}
	a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(msg)
	cmds = append(cmds, cmd)
	return a, tea.Batch(cmds...)
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

	components = append(components, a.status.View())

	appView := lipgloss.JoinVertical(lipgloss.Top, components...)

	if a.showPermissions {
		overlay := a.permissions.View()
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

	if a.showHelp {
		bindings := layout.KeyMapToSlice(keys)
		if p, ok := a.pages[a.currentPage].(layout.Bindings); ok {
			bindings = append(bindings, p.BindingKeys()...)
		}
		if a.showPermissions {
			bindings = append(bindings, a.permissions.BindingKeys()...)
		}
		if a.currentPage == page.LogsPage {
			bindings = append(bindings, logsKeyReturnKey)
		}

		a.help.SetBindings(bindings)

		overlay := a.help.View()
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

	if a.showQuit {
		overlay := a.quit.View()
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
	startPage := page.ChatPage
	return &appModel{
		currentPage: startPage,
		loadedPages: make(map[page.PageID]bool),
		status:      core.NewStatusCmp(app.LSPClients),
		help:        dialog.NewHelpCmp(),
		quit:        dialog.NewQuitCmp(),
		permissions: dialog.NewPermissionDialogCmp(),
		app:         app,
		pages: map[page.PageID]tea.Model{
			page.ChatPage: page.NewChatPage(app),
			page.LogsPage: page.NewLogsPage(),
		},
	}
}
