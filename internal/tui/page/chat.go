package page

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/message"
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

func (p *chatPage) Init() tea.Cmd {
	return p.layout.Init()
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

func (p *chatPage) sendMessage(text string) tea.Cmd {
	var cmds []tea.Cmd
	if p.session.ID == "" {
		session, err := p.app.Sessions.Create("New Session")
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
	// TODO: actually call agent
	p.app.Messages.Create(p.session.ID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{
				Text: text,
			},
		},
	})
	return tea.Batch(cmds...)
}

func (p *chatPage) View() string {
	return p.layout.View()
}

func NewChatPage(app *app.App) tea.Model {
	messagesContainer := layout.NewContainer(
		chat.NewMessagesCmp(app),
		layout.WithPadding(1, 1, 1, 1),
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
