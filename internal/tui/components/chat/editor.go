package chat

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/tui/layout"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
	"github.com/kujtimiihoxha/termai/internal/tui/util"
)

type editorCmp struct {
	textarea     textarea.Model
	agentWorking bool
}

type focusedEditorKeyMaps struct {
	Send key.Binding
	Blur key.Binding
}

type bluredEditorKeyMaps struct {
	Send  key.Binding
	Focus key.Binding
}

var focusedKeyMaps = focusedEditorKeyMaps{
	Send: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "send message"),
	),
	Blur: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "blur editor"),
	),
}

var bluredKeyMaps = bluredEditorKeyMaps{
	Send: key.NewBinding(
		key.WithKeys("ctrl+s", "enter"),
		key.WithHelp("ctrl+s/enter", "send message"),
	),
	Focus: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "focus editor"),
	),
}

func (m *editorCmp) Init() tea.Cmd {
	return textarea.Blink
}

func (m *editorCmp) send() tea.Cmd {
	if m.agentWorking {
		return util.ReportWarn("Agent is working, please wait...")
	}

	value := m.textarea.Value()
	m.textarea.Reset()
	m.textarea.Blur()
	if value == "" {
		return nil
	}
	return tea.Batch(
		util.CmdHandler(SendMsg{
			Text: value,
		}),
		util.CmdHandler(AgentWorkingMsg(true)),
		util.CmdHandler(EditorFocusMsg(false)),
	)
}

func (m *editorCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case AgentWorkingMsg:
		m.agentWorking = bool(msg)
	case tea.KeyMsg:
		if key.Matches(msg, focusedKeyMaps.Send) {
			return m, m.send()
		}
		if key.Matches(msg, bluredKeyMaps.Send) {
			return m, m.send()
		}
		if key.Matches(msg, focusedKeyMaps.Blur) {
			m.textarea.Blur()
			return m, util.CmdHandler(EditorFocusMsg(false))
		}
		if key.Matches(msg, bluredKeyMaps.Focus) {
			if !m.textarea.Focused() {
				m.textarea.Focus()
				return m, tea.Batch(textarea.Blink, util.CmdHandler(EditorFocusMsg(true)))
			}
		}
	}
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m *editorCmp) View() string {
	style := lipgloss.NewStyle().Padding(0, 0, 0, 1).Bold(true)

	return lipgloss.JoinHorizontal(lipgloss.Top, style.Render(">"), m.textarea.View())
}

func (m *editorCmp) SetSize(width, height int) {
	m.textarea.SetWidth(width - 3) // account for the prompt and padding right
	m.textarea.SetHeight(height)
}

func (m *editorCmp) GetSize() (int, int) {
	return m.textarea.Width(), m.textarea.Height()
}

func (m *editorCmp) BindingKeys() []key.Binding {
	bindings := layout.KeyMapToSlice(m.textarea.KeyMap)
	if m.textarea.Focused() {
		bindings = append(bindings, layout.KeyMapToSlice(focusedKeyMaps)...)
	} else {
		bindings = append(bindings, layout.KeyMapToSlice(bluredKeyMaps)...)
	}
	return bindings
}

func NewEditorCmp() tea.Model {
	ti := textarea.New()
	ti.Prompt = " "
	ti.ShowLineNumbers = false
	ti.BlurredStyle.Base = ti.BlurredStyle.Base.Background(styles.Background)
	ti.BlurredStyle.CursorLine = ti.BlurredStyle.CursorLine.Background(styles.Background)
	ti.BlurredStyle.Placeholder = ti.BlurredStyle.Placeholder.Background(styles.Background)
	ti.BlurredStyle.Text = ti.BlurredStyle.Text.Background(styles.Background)

	ti.FocusedStyle.Base = ti.FocusedStyle.Base.Background(styles.Background)
	ti.FocusedStyle.CursorLine = ti.FocusedStyle.CursorLine.Background(styles.Background)
	ti.FocusedStyle.Placeholder = ti.FocusedStyle.Placeholder.Background(styles.Background)
	ti.FocusedStyle.Text = ti.BlurredStyle.Text.Background(styles.Background)
	ti.CharLimit = -1
	ti.Focus()
	return &editorCmp{
		textarea: ti,
	}
}
