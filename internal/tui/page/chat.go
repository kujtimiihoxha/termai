package page

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/session"
	"github.com/kujtimiihoxha/termai/internal/tui/components/chat"
	"github.com/kujtimiihoxha/termai/internal/tui/layout"
	"github.com/kujtimiihoxha/termai/internal/tui/util"
)

var ChatPage PageID = "chat"

type chatPage struct {
	app     *app.App
	layout  layout.SplitPaneLayout
	session session.Session
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
	// TODO: remove
	cmds := []tea.Cmd{
		p.layout.Init(),
	}

	sessions, _ := p.app.Sessions.List(context.Background())
	if len(sessions) > 0 {
		p.session = sessions[0]
		cmd := p.setSidebar()
		cmds = append(cmds, util.CmdHandler(chat.SessionSelectedMsg(p.session)), cmd)
	}
	return tea.Batch(
		cmds...,
	)
}

func (p *chatPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.layout.SetSize(msg.Width, msg.Height)
	case chat.SendMsg:
		cmd := p.sendMessage(msg.Text)
		if cmd != nil {
			return p, cmd
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keyMap.NewSession):
			p.session = session.Session{}
			p.clearSidebar()
			return p, util.CmdHandler(chat.SessionClearedMsg{})
		}
	}
	u, cmd := p.layout.Update(msg)
	p.layout = u.(layout.SplitPaneLayout)
	if cmd != nil {
		return p, cmd
	}
	return p, nil
}

func (p *chatPage) setSidebar() tea.Cmd {
	sidebarContainer := layout.NewContainer(
		chat.NewSidebarCmp(p.session),
		layout.WithPadding(1, 1, 1, 1),
	)
	p.layout.SetRightPanel(sidebarContainer)
	width, height := p.layout.GetSize()
	p.layout.SetSize(width, height)
	return sidebarContainer.Init()
}

func (p *chatPage) clearSidebar() {
	p.layout.SetRightPanel(nil)
	width, height := p.layout.GetSize()
	p.layout.SetSize(width, height)
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

	p.app.CoderAgent.Generate(context.Background(), p.session.ID, text)
	return tea.Batch(cmds...)
}

func (p *chatPage) View() string {
	return p.layout.View()
}

func NewChatPage(app *app.App) tea.Model {
	messagesContainer := layout.NewContainer(
		chat.NewMessagesCmp(app),
		layout.WithPadding(1, 1, 0, 1),
	)

	editorContainer := layout.NewContainer(
		chat.NewEditorCmp(),
		layout.WithBorder(true, false, false, false),
	)
	return &chatPage{
		app: app,
		layout: layout.NewSplitPane(
			layout.WithLeftPanel(messagesContainer),
			layout.WithBottomPanel(editorContainer),
		),
	}
}
