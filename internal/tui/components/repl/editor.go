package repl

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/llm/agent"
	"github.com/kujtimiihoxha/termai/internal/tui/layout"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
	"github.com/kujtimiihoxha/termai/internal/tui/util"
	"github.com/kujtimiihoxha/vimtea"
	"golang.org/x/net/context"
)

type EditorCmp interface {
	tea.Model
	layout.Focusable
	layout.Sizeable
	layout.Bordered
	layout.Bindings
}

type editorCmp struct {
	app           *app.App
	editor        vimtea.Editor
	editorMode    vimtea.EditorMode
	sessionID     string
	focused       bool
	width         int
	height        int
	cancelMessage context.CancelFunc
}

type editorKeyMap struct {
	SendMessage    key.Binding
	SendMessageI   key.Binding
	CancelMessage  key.Binding
	InsertMode     key.Binding
	NormaMode      key.Binding
	VisualMode     key.Binding
	VisualLineMode key.Binding
}

var editorKeyMapValue = editorKeyMap{
	SendMessage: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "send message normal mode"),
	),
	SendMessageI: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "send message insert mode"),
	),
	CancelMessage: key.NewBinding(
		key.WithKeys("ctrl+x"),
		key.WithHelp("ctrl+x", "cancel current message"),
	),
	InsertMode: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "insert mode"),
	),
	NormaMode: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "normal mode"),
	),
	VisualMode: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "visual mode"),
	),
	VisualLineMode: key.NewBinding(
		key.WithKeys("V"),
		key.WithHelp("V", "visual line mode"),
	),
}

func (m *editorCmp) Init() tea.Cmd {
	return m.editor.Init()
}

func (m *editorCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case vimtea.EditorModeMsg:
		m.editorMode = msg.Mode
	case SelectedSessionMsg:
		if msg.SessionID != m.sessionID {
			m.sessionID = msg.SessionID
		}
	}
	if m.IsFocused() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, editorKeyMapValue.SendMessage):
				if m.editorMode == vimtea.ModeNormal {
					return m, m.Send()
				}
			case key.Matches(msg, editorKeyMapValue.SendMessageI):
				if m.editorMode == vimtea.ModeInsert {
					return m, m.Send()
				}
			case key.Matches(msg, editorKeyMapValue.CancelMessage):
				return m, m.Cancel()
			}
		}
		u, cmd := m.editor.Update(msg)
		m.editor = u.(vimtea.Editor)
		return m, cmd
	}
	return m, nil
}

func (m *editorCmp) Blur() tea.Cmd {
	m.focused = false
	return nil
}

func (m *editorCmp) BorderText() map[layout.BorderPosition]string {
	title := "New Message"
	if m.focused {
		title = lipgloss.NewStyle().Foreground(styles.Primary).Render(title)
	}
	return map[layout.BorderPosition]string{
		layout.BottomLeftBorder: title,
	}
}

func (m *editorCmp) Focus() tea.Cmd {
	m.focused = true
	return m.editor.Tick()
}

func (m *editorCmp) GetSize() (int, int) {
	return m.width, m.height
}

func (m *editorCmp) IsFocused() bool {
	return m.focused
}

func (m *editorCmp) SetSize(width int, height int) {
	m.width = width
	m.height = height
	m.editor.SetSize(width, height)
}

func (m *editorCmp) Cancel() tea.Cmd {
	if m.cancelMessage == nil {
		return util.ReportWarn("No message to cancel")
	}

	m.cancelMessage()
	m.cancelMessage = nil
	return util.ReportWarn("Message cancelled")
}

func (m *editorCmp) Send() tea.Cmd {
	if m.cancelMessage != nil {
		return util.ReportWarn("Assistant is still working on the previous message")
	}

	messages, err := m.app.Messages.List(context.Background(), m.sessionID)
	if err != nil {
		return util.ReportError(err)
	}
	if hasUnfinishedMessages(messages) {
		return util.ReportWarn("Assistant is still working on the previous message")
	}

	a, err := agent.NewCoderAgent(m.app)
	if err != nil {
		return util.ReportError(err)
	}

	content := strings.Join(m.editor.GetBuffer().Lines(), "\n")
	if len(content) == 0 {
		return util.ReportWarn("Message is empty")
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelMessage = cancel
	go func() {
		defer cancel()
		a.Generate(ctx, m.sessionID, content)
		m.cancelMessage = nil
	}()

	return m.editor.Reset()
}

func (m *editorCmp) View() string {
	return m.editor.View()
}

func (m *editorCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(editorKeyMapValue)
}

func NewEditorCmp(app *app.App) EditorCmp {
	editor := vimtea.NewEditor(
		vimtea.WithFileName("message.md"),
	)
	return &editorCmp{
		app:    app,
		editor: editor,
	}
}
