package page

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kujtimiihoxha/opencode/internal/app"
	"github.com/kujtimiihoxha/opencode/internal/session"
	"github.com/kujtimiihoxha/opencode/internal/tui/components/chat"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
	"github.com/kujtimiihoxha/opencode/internal/tui/util"
)

var ChatPage PageID = "chat"

type chatPage struct {
	app         *app.App
	editor      layout.Container
	messages    layout.Container
	layout      layout.SplitPaneLayout
	session     session.Session
	editingMode bool
}

type ChatKeyMap struct {
	NewSession key.Binding
	Cancel     key.Binding
}

var keyMap = ChatKeyMap{
	NewSession: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "new session"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("ctrl+x"),
		key.WithHelp("ctrl+x", "cancel"),
	),
}

func (p *chatPage) Init() tea.Cmd {
	cmds := []tea.Cmd{
		p.layout.Init(),
	}

	sessions, _ := p.app.Sessions.List(context.Background())
	if len(sessions) > 0 {
		p.session = sessions[0]
		cmd := p.setSidebar()
		cmds = append(cmds, util.CmdHandler(chat.SessionSelectedMsg(p.session)), cmd)
	}
	return tea.Batch(cmds...)
}

func (p *chatPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := p.layout.SetSize(msg.Width, msg.Height)
		cmds = append(cmds, cmd)
	case chat.SendMsg:
		cmd := p.sendMessage(msg.Text)
		if cmd != nil {
			return p, cmd
		}
	case chat.EditorFocusMsg:
		p.editingMode = bool(msg)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keyMap.NewSession):
			p.session = session.Session{}
			return p, tea.Batch(
				p.clearSidebar(),
				util.CmdHandler(chat.SessionClearedMsg{}),
			)
		case key.Matches(msg, keyMap.Cancel):
			if p.session.ID != "" {
				// Cancel the current session's generation process
				// This allows users to interrupt long-running operations
				p.app.CoderAgent.Cancel(p.session.ID)
				return p, nil
			}
		}
	}
	u, cmd := p.layout.Update(msg)
	cmds = append(cmds, cmd)
	p.layout = u.(layout.SplitPaneLayout)
	return p, tea.Batch(cmds...)
}

func (p *chatPage) setSidebar() tea.Cmd {
	sidebarContainer := layout.NewContainer(
		chat.NewSidebarCmp(p.session, p.app.History),
		layout.WithPadding(1, 1, 1, 1),
	)
	return tea.Batch(p.layout.SetRightPanel(sidebarContainer), sidebarContainer.Init())
}

func (p *chatPage) clearSidebar() tea.Cmd {
	return p.layout.SetRightPanel(nil)
}

func (p *chatPage) sendMessage(text string) tea.Cmd {
	var cmds []tea.Cmd
	if p.session.ID == "" {
		session, err := p.app.Sessions.Create(context.Background(), "New Session")
		if err != nil {
			return util.ReportError(err)
		}

		p.session = session
		cmd := p.setSidebar()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, util.CmdHandler(chat.SessionSelectedMsg(session)))
	}

	p.app.CoderAgent.Run(context.Background(), p.session.ID, text)
	return tea.Batch(cmds...)
}

func (p *chatPage) SetSize(width, height int) tea.Cmd {
	return p.layout.SetSize(width, height)
}

func (p *chatPage) GetSize() (int, int) {
	return p.layout.GetSize()
}

func (p *chatPage) View() string {
	return p.layout.View()
}

func (p *chatPage) BindingKeys() []key.Binding {
	bindings := layout.KeyMapToSlice(keyMap)
	if p.editingMode {
		bindings = append(bindings, p.editor.BindingKeys()...)
	} else {
		bindings = append(bindings, p.messages.BindingKeys()...)
	}
	return bindings
}

func NewChatPage(app *app.App) tea.Model {
	messagesContainer := layout.NewContainer(
		chat.NewMessagesCmp(app),
		layout.WithPadding(1, 1, 0, 1),
	)

	editorContainer := layout.NewContainer(
		chat.NewEditorCmp(app),
		layout.WithBorder(true, false, false, false),
	)
	return &chatPage{
		app:         app,
		editor:      editorContainer,
		messages:    messagesContainer,
		editingMode: true,
		layout: layout.NewSplitPane(
			layout.WithLeftPanel(messagesContainer),
			layout.WithBottomPanel(editorContainer),
		),
	}
}
