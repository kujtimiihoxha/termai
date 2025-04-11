package page

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/tui/components/chat"
	"github.com/kujtimiihoxha/termai/internal/tui/layout"
)

var ChatPage PageID = "chat"

func NewChatPage(app *app.App) tea.Model {
	messagesContainer := layout.NewContainer(
		chat.NewMessagesCmp(),
		layout.WithPadding(1, 1, 1, 1),
	)
	sidebarContainer := layout.NewContainer(
		chat.NewSidebarCmp(),
		layout.WithPadding(1, 1, 1, 1),
	)
	editorContainer := layout.NewContainer(
		chat.NewEditorCmp(),
		layout.WithBorder(true, false, false, false),
	)
	return layout.NewSplitPane(
		layout.WithRightPanel(sidebarContainer),
		layout.WithLeftPanel(messagesContainer),
		layout.WithBottomPanel(editorContainer),
	)
}
