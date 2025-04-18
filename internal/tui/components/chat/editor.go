package chat

import (
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/opencode/internal/app"
	"github.com/kujtimiihoxha/opencode/internal/session"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
	"github.com/kujtimiihoxha/opencode/internal/tui/util"
)

type editorCmp struct {
	app      *app.App
	session  session.Session
	textarea textarea.Model
}

type focusedEditorKeyMaps struct {
	Send       key.Binding
	OpenEditor key.Binding
	Blur       key.Binding
}

type bluredEditorKeyMaps struct {
	Send       key.Binding
	Focus      key.Binding
	OpenEditor key.Binding
}

var focusedKeyMaps = focusedEditorKeyMaps{
	Send: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "send message"),
	),
	Blur: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "focus messages"),
	),
	OpenEditor: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "open editor"),
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
	OpenEditor: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "open editor"),
	),
}

func openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}

	tmpfile, err := os.CreateTemp("", "msg_*.md")
	if err != nil {
		return util.ReportError(err)
	}
	tmpfile.Close()
	c := exec.Command(editor, tmpfile.Name()) //nolint:gosec
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return util.ReportError(err)
		}
		content, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			return util.ReportError(err)
		}
		os.Remove(tmpfile.Name())
		return SendMsg{
			Text: string(content),
		}
	})
}

func (m *editorCmp) Init() tea.Cmd {
	return textarea.Blink
}

func (m *editorCmp) send() tea.Cmd {
	if m.app.CoderAgent.IsSessionBusy(m.session.ID) {
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
		util.CmdHandler(EditorFocusMsg(false)),
	)
}

func (m *editorCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case SessionSelectedMsg:
		if msg.ID != m.session.ID {
			m.session = msg
		}
		return m, nil
	case tea.KeyMsg:
		if key.Matches(msg, focusedKeyMaps.OpenEditor) {
			m.textarea.Blur()
			return m, openEditor()
		}
		// if the key does not match any binding, return
		if m.textarea.Focused() && key.Matches(msg, focusedKeyMaps.Send) {
			return m, m.send()
		}
		if !m.textarea.Focused() && key.Matches(msg, bluredKeyMaps.Send) {
			return m, m.send()
		}
		if m.textarea.Focused() && key.Matches(msg, focusedKeyMaps.Blur) {
			m.textarea.Blur()
			return m, util.CmdHandler(EditorFocusMsg(false))
		}
		if !m.textarea.Focused() && key.Matches(msg, bluredKeyMaps.Focus) {
			m.textarea.Focus()
			return m, tea.Batch(textarea.Blink, util.CmdHandler(EditorFocusMsg(true)))
		}
	}
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m *editorCmp) View() string {
	style := lipgloss.NewStyle().Padding(0, 0, 0, 1).Bold(true)

	return lipgloss.JoinHorizontal(lipgloss.Top, style.Render(">"), m.textarea.View())
}

func (m *editorCmp) SetSize(width, height int) tea.Cmd {
	m.textarea.SetWidth(width - 3) // account for the prompt and padding right
	m.textarea.SetHeight(height)
	return nil
}

func (m *editorCmp) GetSize() (int, int) {
	return m.textarea.Width(), m.textarea.Height()
}

func (m *editorCmp) BindingKeys() []key.Binding {
	bindings := []key.Binding{}
	if m.textarea.Focused() {
		bindings = append(bindings, layout.KeyMapToSlice(focusedKeyMaps)...)
	} else {
		bindings = append(bindings, layout.KeyMapToSlice(bluredKeyMaps)...)
	}

	bindings = append(bindings, layout.KeyMapToSlice(m.textarea.KeyMap)...)
	return bindings
}

func NewEditorCmp(app *app.App) tea.Model {
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
		app:      app,
		textarea: ti,
	}
}
