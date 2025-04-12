package chat

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/llm/agent"
	"github.com/kujtimiihoxha/termai/internal/llm/models"
	"github.com/kujtimiihoxha/termai/internal/llm/tools"
	"github.com/kujtimiihoxha/termai/internal/message"
	"github.com/kujtimiihoxha/termai/internal/pubsub"
	"github.com/kujtimiihoxha/termai/internal/session"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
	"github.com/kujtimiihoxha/termai/internal/tui/util"
)

type uiMessageType int

const (
	userMessageType uiMessageType = iota
	assistantMessageType
	toolMessageType
)

type uiMessage struct {
	ID          string
	messageType uiMessageType
	position    int
	height      int
	content     string
}

type messagesCmp struct {
	app           *app.App
	width, height int
	writingMode   bool
	viewport      viewport.Model
	session       session.Session
	messages      []message.Message
	uiMessages    []uiMessage
	currentMsgID  string
	renderer      *glamour.TermRenderer
	focusRenderer *glamour.TermRenderer
	cachedContent map[string]string
	agentWorking  bool
	spinner       spinner.Model
	needsRerender bool
	lastViewport  string
}

func (m *messagesCmp) Init() tea.Cmd {
	return tea.Batch(m.viewport.Init())
}

func (m *messagesCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case AgentWorkingMsg:
		m.agentWorking = bool(msg)
		if m.agentWorking {
			cmds = append(cmds, m.spinner.Tick)
		}
	case EditorFocusMsg:
		m.writingMode = bool(msg)
	case SessionSelectedMsg:
		if msg.ID != m.session.ID {
			cmd := m.SetSession(msg)
			m.needsRerender = true
			return m, cmd
		}
		return m, nil
	case SessionClearedMsg:
		m.session = session.Session{}
		m.messages = make([]message.Message, 0)
		m.currentMsgID = ""
		m.needsRerender = true
		return m, nil

	case tea.KeyMsg:
		if m.writingMode {
			return m, nil
		}
	case pubsub.Event[message.Message]:
		if msg.Type == pubsub.CreatedEvent {
			if msg.Payload.SessionID == m.session.ID {
				// check if message exists

				messageExists := false
				for _, v := range m.messages {
					if v.ID == msg.Payload.ID {
						messageExists = true
						break
					}
				}

				if !messageExists {
					m.messages = append(m.messages, msg.Payload)
					delete(m.cachedContent, m.currentMsgID)
					m.currentMsgID = msg.Payload.ID
					m.needsRerender = true
				}
			}
			for _, v := range m.messages {
				for _, c := range v.ToolCalls() {
					// the message is being added to the session of a tool called
					if c.ID == msg.Payload.SessionID {
						m.needsRerender = true
					}
				}
			}
		} else if msg.Type == pubsub.UpdatedEvent && msg.Payload.SessionID == m.session.ID {
			for i, v := range m.messages {
				if v.ID == msg.Payload.ID {
					if !m.messages[i].IsFinished() && msg.Payload.IsFinished() && msg.Payload.FinishReason() == "end_turn" || msg.Payload.FinishReason() == "canceled" {
						cmds = append(cmds, util.CmdHandler(AgentWorkingMsg(false)))
					}
					m.messages[i] = msg.Payload
					delete(m.cachedContent, msg.Payload.ID)
					m.needsRerender = true
					break
				}
			}
		}
	}
	if m.agentWorking {
		u, cmd := m.spinner.Update(msg)
		m.spinner = u
		cmds = append(cmds, cmd)
	}
	oldPos := m.viewport.YPosition
	u, cmd := m.viewport.Update(msg)
	m.viewport = u
	m.needsRerender = m.needsRerender || m.viewport.YPosition != oldPos
	cmds = append(cmds, cmd)
	if m.needsRerender {
		m.renderView()
		if len(m.messages) > 0 {
			if msg, ok := msg.(pubsub.Event[message.Message]); ok {
				if (msg.Type == pubsub.CreatedEvent) ||
					(msg.Type == pubsub.UpdatedEvent && msg.Payload.ID == m.messages[len(m.messages)-1].ID) {
					m.viewport.GotoBottom()
				}
			}
		}
		m.needsRerender = false
	}
	return m, tea.Batch(cmds...)
}

func (m *messagesCmp) renderSimpleMessage(msg message.Message, info ...string) string {
	if v, ok := m.cachedContent[msg.ID]; ok {
		return v
	}
	style := styles.BaseStyle.
		Width(m.width).
		BorderLeft(true).
		Foreground(styles.ForgroundDim).
		BorderForeground(styles.ForgroundDim).
		BorderStyle(lipgloss.ThickBorder())

	renderer := m.renderer
	if msg.ID == m.currentMsgID {
		style = style.
			Foreground(styles.Forground).
			BorderForeground(styles.Blue).
			BorderStyle(lipgloss.ThickBorder())
		renderer = m.focusRenderer
	}
	c, _ := renderer.Render(msg.Content().String())
	parts := []string{
		styles.ForceReplaceBackgroundWithLipgloss(c, styles.Background),
	}
	// remove newline at the end
	parts[0] = strings.TrimSuffix(parts[0], "\n")
	if len(info) > 0 {
		parts = append(parts, info...)
	}
	rendered := style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			parts...,
		),
	)
	m.cachedContent[msg.ID] = rendered
	return rendered
}

func formatTimeDifference(unixTime1, unixTime2 int64) string {
	diffSeconds := float64(math.Abs(float64(unixTime2 - unixTime1)))

	if diffSeconds < 60 {
		return fmt.Sprintf("%.1fs", diffSeconds)
	}

	minutes := int(diffSeconds / 60)
	seconds := int(diffSeconds) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

func (m *messagesCmp) renderToolCall(toolCall message.ToolCall, isNested bool) string {
	key := ""
	value := ""
	switch toolCall.Name {
	// TODO: add result data to the tools
	case agent.AgentToolName:
		key = "Task"
		var params agent.AgentParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.Prompt
	// TODO: handle nested calls
	case tools.BashToolName:
		key = "Bash"
		var params tools.BashParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.Command
	case tools.EditToolName:
		key = "Edit"
		var params tools.EditParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.FilePath
	case tools.FetchToolName:
		key = "Fetch"
		var params tools.FetchParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.URL
	case tools.GlobToolName:
		key = "Glob"
		var params tools.GlobParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		if params.Path == "" {
			params.Path = "."
		}
		value = fmt.Sprintf("%s (%s)", params.Pattern, params.Path)
	case tools.GrepToolName:
		key = "Grep"
		var params tools.GrepParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		if params.Path == "" {
			params.Path = "."
		}
		value = fmt.Sprintf("%s (%s)", params.Pattern, params.Path)
	case tools.LSToolName:
		key = "Ls"
		var params tools.LSParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		if params.Path == "" {
			params.Path = "."
		}
		value = params.Path
	case tools.SourcegraphToolName:
		key = "Sourcegraph"
		var params tools.SourcegraphParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.Query
	case tools.ViewToolName:
		key = "View"
		var params tools.ViewParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.FilePath
	case tools.WriteToolName:
		key = "Write"
		var params tools.WriteParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.FilePath
	default:
		key = toolCall.Name
		var params map[string]any
		json.Unmarshal([]byte(toolCall.Input), &params)
		jsonData, _ := json.Marshal(params)
		value = string(jsonData)
	}

	style := styles.BaseStyle.
		Width(m.width).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		PaddingLeft(1).
		BorderForeground(styles.Yellow)

	keyStyle := styles.BaseStyle.
		Foreground(styles.ForgroundDim)
	valyeStyle := styles.BaseStyle.
		Foreground(styles.Forground)

	if isNested {
		valyeStyle = valyeStyle.Foreground(styles.ForgroundMid)
	}
	keyValye := keyStyle.Render(
		fmt.Sprintf("%s: ", key),
	)
	if !isNested {
		value = valyeStyle.
			Width(m.width - lipgloss.Width(keyValye) - 2).
			Render(
				ansi.Truncate(
					value,
					m.width-lipgloss.Width(keyValye)-2,
					"...",
				),
			)
	} else {
		keyValye = keyStyle.Render(
			fmt.Sprintf(" └ %s: ", key),
		)
		value = valyeStyle.
			Width(m.width - lipgloss.Width(keyValye) - 2).
			Render(
				ansi.Truncate(
					value,
					m.width-lipgloss.Width(keyValye)-2,
					"...",
				),
			)
	}

	innerToolCalls := make([]string, 0)
	if toolCall.Name == agent.AgentToolName {
		messages, _ := m.app.Messages.List(toolCall.ID)
		toolCalls := make([]message.ToolCall, 0)
		for _, v := range messages {
			toolCalls = append(toolCalls, v.ToolCalls()...)
		}
		for _, v := range toolCalls {
			call := m.renderToolCall(v, true)
			innerToolCalls = append(innerToolCalls, call)
		}
	}

	if isNested {
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			keyValye,
			value,
		)
	}
	callContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		keyValye,
		value,
	)
	callContent = strings.ReplaceAll(callContent, "\n", "")
	if len(innerToolCalls) > 0 {
		callContent = lipgloss.JoinVertical(
			lipgloss.Left,
			callContent,
			lipgloss.JoinVertical(
				lipgloss.Left,
				innerToolCalls...,
			),
		)
	}
	return style.Render(callContent)
}

func (m *messagesCmp) renderAssistantMessage(msg message.Message) []uiMessage {
	// find the user message that is before this assistant message
	var userMsg message.Message
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == message.User {
			userMsg = m.messages[i]
			break
		}
	}
	messages := make([]uiMessage, 0)
	if msg.Content().String() != "" {
		info := make([]string, 0)
		if msg.IsFinished() && msg.FinishReason() == "end_turn" {
			finish := msg.FinishPart()
			took := formatTimeDifference(userMsg.CreatedAt, finish.Time)

			info = append(info, styles.BaseStyle.Width(m.width-1).Foreground(styles.ForgroundDim).Render(
				fmt.Sprintf(" %s (%s)", models.SupportedModels[msg.Model].Name, took),
			))
		}
		content := m.renderSimpleMessage(msg, info...)
		messages = append(messages, uiMessage{
			messageType: assistantMessageType,
			position:    0, // gets updated in renderView
			height:      lipgloss.Height(content),
			content:     content,
		})
	}
	for _, v := range msg.ToolCalls() {
		content := m.renderToolCall(v, false)
		messages = append(messages,
			uiMessage{
				messageType: toolMessageType,
				position:    0, // gets updated in renderView
				height:      lipgloss.Height(content),
				content:     content,
			},
		)
	}

	return messages
}

func (m *messagesCmp) renderView() {
	m.uiMessages = make([]uiMessage, 0)
	pos := 0

	for _, v := range m.messages {
		switch v.Role {
		case message.User:
			content := m.renderSimpleMessage(v)
			m.uiMessages = append(m.uiMessages, uiMessage{
				messageType: userMessageType,
				position:    pos,
				height:      lipgloss.Height(content),
				content:     content,
			})
			pos += lipgloss.Height(content) + 1 // + 1 for spacing
		case message.Assistant:
			assistantMessages := m.renderAssistantMessage(v)
			for _, msg := range assistantMessages {
				msg.position = pos
				m.uiMessages = append(m.uiMessages, msg)
				pos += msg.height + 1 // + 1 for spacing
			}

		}
	}

	messages := make([]string, 0)
	for _, v := range m.uiMessages {
		messages = append(messages, v.content,
			styles.BaseStyle.
				Width(m.width).
				Render(
					"",
				),
		)
	}
	m.viewport.SetContent(
		styles.BaseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					messages...,
				),
			),
	)
}

func (m *messagesCmp) View() string {
	if len(m.messages) == 0 {
		content := styles.BaseStyle.
			Width(m.width).
			Height(m.height - 1).
			Render(
				m.initialScreen(),
			)

		return styles.BaseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					content,
					m.help(),
				),
			)
	}

	return styles.BaseStyle.
		Width(m.width).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				m.viewport.View(),
				m.help(),
			),
		)
}

func (m *messagesCmp) help() string {
	text := ""

	if m.agentWorking {
		text += styles.BaseStyle.Foreground(styles.PrimaryColor).Bold(true).Render(
			fmt.Sprintf("%s %s ", m.spinner.View(), "Generating..."),
		)
	}
	if m.writingMode {
		text += lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.BaseStyle.Foreground(styles.ForgroundDim).Bold(true).Render("press "),
			styles.BaseStyle.Foreground(styles.Forground).Bold(true).Render("esc"),
			styles.BaseStyle.Foreground(styles.ForgroundDim).Bold(true).Render(" to exit writing mode"),
		)
	} else {
		text += lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.BaseStyle.Foreground(styles.ForgroundDim).Bold(true).Render("press "),
			styles.BaseStyle.Foreground(styles.Forground).Bold(true).Render("i"),
			styles.BaseStyle.Foreground(styles.ForgroundDim).Bold(true).Render(" to start writing"),
		)
	}

	return styles.BaseStyle.
		Width(m.width).
		Render(text)
}

func (m *messagesCmp) initialScreen() string {
	return styles.BaseStyle.Width(m.width).Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			header(m.width),
			"",
			lspsConfigured(m.width),
		),
	)
}

func (m *messagesCmp) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 1
	focusRenderer, _ := glamour.NewTermRenderer(
		glamour.WithStyles(styles.MarkdownTheme(true)),
		glamour.WithWordWrap(width-1),
	)
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStyles(styles.MarkdownTheme(false)),
		glamour.WithWordWrap(width-1),
	)
	m.focusRenderer = focusRenderer
	// clear the cached content
	for k := range m.cachedContent {
		delete(m.cachedContent, k)
	}
	m.renderer = renderer
	if len(m.messages) > 0 {
		m.renderView()
		m.viewport.GotoBottom()
	}
}

func (m *messagesCmp) GetSize() (int, int) {
	return m.width, m.height
}

func (m *messagesCmp) SetSession(session session.Session) tea.Cmd {
	m.session = session
	messages, err := m.app.Messages.List(session.ID)
	if err != nil {
		return util.ReportError(err)
	}
	m.messages = messages
	m.currentMsgID = m.messages[len(m.messages)-1].ID
	m.needsRerender = true
	return nil
}

func NewMessagesCmp(app *app.App) tea.Model {
	focusRenderer, _ := glamour.NewTermRenderer(
		glamour.WithStyles(styles.MarkdownTheme(true)),
		glamour.WithWordWrap(80),
	)
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStyles(styles.MarkdownTheme(false)),
		glamour.WithWordWrap(80),
	)

	s := spinner.New()
	s.Spinner = spinner.Pulse
	return &messagesCmp{
		app:           app,
		writingMode:   true,
		cachedContent: make(map[string]string),
		viewport:      viewport.New(0, 0),
		focusRenderer: focusRenderer,
		renderer:      renderer,
		spinner:       s,
	}
}
