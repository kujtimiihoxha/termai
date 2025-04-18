package chat

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/opencode/internal/app"
	"github.com/kujtimiihoxha/opencode/internal/logging"
	"github.com/kujtimiihoxha/opencode/internal/message"
	"github.com/kujtimiihoxha/opencode/internal/pubsub"
	"github.com/kujtimiihoxha/opencode/internal/session"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
	"github.com/kujtimiihoxha/opencode/internal/tui/util"
)

type messagesCmp struct {
	app           *app.App
	width, height int
	writingMode   bool
	viewport      viewport.Model
	session       session.Session
	messages      []message.Message
	uiMessages    []uiMessage
	currentMsgID  string
	mutex         sync.Mutex
	cachedContent map[string][]uiMessage
	spinner       spinner.Model
	rendering     bool
}
type renderFinishedMsg struct{}

func (m *messagesCmp) Init() tea.Cmd {
	return tea.Batch(m.viewport.Init())
}

func (m *messagesCmp) preloadSessions() tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.app.Sessions.List(context.Background())
		if err != nil {
			return util.ReportError(err)()
		}
		if len(sessions) == 0 {
			return nil
		}
		if len(sessions) > 20 {
			sessions = sessions[:20]
		}
		for _, s := range sessions {
			messages, err := m.app.Messages.List(context.Background(), s.ID)
			if err != nil {
				return util.ReportError(err)()
			}
			if len(messages) == 0 {
				continue
			}
			m.cacheSessionMessages(messages, m.width)

		}
		logging.Debug("preloaded sessions")

		return nil
	}
}

func (m *messagesCmp) cacheSessionMessages(messages []message.Message, width int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	pos := 0
	if m.width == 0 {
		return
	}
	for inx, msg := range messages {
		switch msg.Role {
		case message.User:
			userMsg := renderUserMessage(
				msg,
				false,
				width,
				pos,
			)
			m.cachedContent[msg.ID] = []uiMessage{userMsg}
			pos += userMsg.height + 1 // + 1 for spacing
		case message.Assistant:
			assistantMessages := renderAssistantMessage(
				msg,
				inx,
				messages,
				m.app.Messages,
				"",
				width,
				pos,
			)
			for _, msg := range assistantMessages {
				pos += msg.height + 1 // + 1 for spacing
			}
			m.cachedContent[msg.ID] = assistantMessages
		}
	}
}

func (m *messagesCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case EditorFocusMsg:
		m.writingMode = bool(msg)
	case SessionSelectedMsg:
		if msg.ID != m.session.ID {
			cmd := m.SetSession(msg)
			return m, cmd
		}
		return m, nil
	case SessionClearedMsg:
		m.session = session.Session{}
		m.messages = make([]message.Message, 0)
		m.currentMsgID = ""
		m.rendering = false
		return m, nil

	case renderFinishedMsg:
		m.rendering = false
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		if m.writingMode {
			return m, nil
		}
	case pubsub.Event[message.Message]:
		needsRerender := false
		if msg.Type == pubsub.CreatedEvent {
			if msg.Payload.SessionID == m.session.ID {

				messageExists := false
				for _, v := range m.messages {
					if v.ID == msg.Payload.ID {
						messageExists = true
						break
					}
				}

				if !messageExists {
					if len(m.messages) > 0 {
						lastMsgID := m.messages[len(m.messages)-1].ID
						delete(m.cachedContent, lastMsgID)
					}

					m.messages = append(m.messages, msg.Payload)
					delete(m.cachedContent, m.currentMsgID)
					m.currentMsgID = msg.Payload.ID
					needsRerender = true
				}
			}
			// There are tool calls from the child task
			for _, v := range m.messages {
				for _, c := range v.ToolCalls() {
					if c.ID == msg.Payload.SessionID {
						delete(m.cachedContent, v.ID)
						needsRerender = true
					}
				}
			}
		} else if msg.Type == pubsub.UpdatedEvent && msg.Payload.SessionID == m.session.ID {
			for i, v := range m.messages {
				if v.ID == msg.Payload.ID {
					m.messages[i] = msg.Payload
					delete(m.cachedContent, msg.Payload.ID)
					needsRerender = true
					break
				}
			}
		}
		if needsRerender {
			m.renderView()
			if len(m.messages) > 0 {
				if (msg.Type == pubsub.CreatedEvent) ||
					(msg.Type == pubsub.UpdatedEvent && msg.Payload.ID == m.messages[len(m.messages)-1].ID) {
					m.viewport.GotoBottom()
				}
			}
		}
	}

	u, cmd := m.viewport.Update(msg)
	m.viewport = u
	cmds = append(cmds, cmd)

	spinner, cmd := m.spinner.Update(msg)
	m.spinner = spinner
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *messagesCmp) IsAgentWorking() bool {
	return m.app.CoderAgent.IsSessionBusy(m.session.ID)
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

func (m *messagesCmp) renderView() {
	m.uiMessages = make([]uiMessage, 0)
	pos := 0

	if m.width == 0 {
		return
	}
	for inx, msg := range m.messages {
		switch msg.Role {
		case message.User:
			if messages, ok := m.cachedContent[msg.ID]; ok {
				m.uiMessages = append(m.uiMessages, messages...)
				continue
			}
			userMsg := renderUserMessage(
				msg,
				msg.ID == m.currentMsgID,
				m.width,
				pos,
			)
			m.uiMessages = append(m.uiMessages, userMsg)
			m.cachedContent[msg.ID] = []uiMessage{userMsg}
			pos += userMsg.height + 1 // + 1 for spacing
		case message.Assistant:
			if messages, ok := m.cachedContent[msg.ID]; ok {
				m.uiMessages = append(m.uiMessages, messages...)
				continue
			}
			assistantMessages := renderAssistantMessage(
				msg,
				inx,
				m.messages,
				m.app.Messages,
				m.currentMsgID,
				m.width,
				pos,
			)
			for _, msg := range assistantMessages {
				m.uiMessages = append(m.uiMessages, msg)
				pos += msg.height + 1 // + 1 for spacing
			}
			m.cachedContent[msg.ID] = assistantMessages
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
	if m.rendering {
		return styles.BaseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					"Loading...",
					m.working(),
					m.help(),
				),
			)
	}
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
					"",
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
				m.working(),
				m.help(),
			),
		)
}

func hasToolsWithoutResponse(messages []message.Message) bool {
	toolCalls := make([]message.ToolCall, 0)
	toolResults := make([]message.ToolResult, 0)
	for _, m := range messages {
		toolCalls = append(toolCalls, m.ToolCalls()...)
		toolResults = append(toolResults, m.ToolResults()...)
	}

	for _, v := range toolCalls {
		found := false
		for _, r := range toolResults {
			if v.ID == r.ToolCallID {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}

	return false
}

func (m *messagesCmp) working() string {
	text := ""
	if m.IsAgentWorking() {
		task := "Thinking..."
		lastMessage := m.messages[len(m.messages)-1]
		if hasToolsWithoutResponse(m.messages) {
			task = "Waiting for tool response..."
		} else if !lastMessage.IsFinished() {
			lastUpdate := lastMessage.UpdatedAt
			currentTime := time.Now().Unix()
			if lastMessage.Content().String() != "" && lastUpdate != 0 && currentTime-lastUpdate > 5 {
				task = "Building tool call..."
			} else if lastMessage.Content().String() == "" {
				task = "Generating..."
			}
			task = ""
		}
		if task != "" {
			text += styles.BaseStyle.Width(m.width).Foreground(styles.PrimaryColor).Bold(true).Render(
				fmt.Sprintf("%s %s ", m.spinner.View(), task),
			)
		}
	}
	return text
}

func (m *messagesCmp) help() string {
	text := ""

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

func (m *messagesCmp) SetSize(width, height int) tea.Cmd {
	if m.width == width && m.height == height {
		return nil
	}
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 2
	m.renderView()
	return m.preloadSessions()
}

func (m *messagesCmp) GetSize() (int, int) {
	return m.width, m.height
}

func (m *messagesCmp) SetSession(session session.Session) tea.Cmd {
	if m.session.ID == session.ID {
		return nil
	}
	m.rendering = true
	return func() tea.Msg {
		m.session = session
		messages, err := m.app.Messages.List(context.Background(), session.ID)
		if err != nil {
			return util.ReportError(err)
		}
		m.messages = messages
		m.currentMsgID = m.messages[len(m.messages)-1].ID
		delete(m.cachedContent, m.currentMsgID)
		m.renderView()
		return renderFinishedMsg{}
	}
}

func (m *messagesCmp) BindingKeys() []key.Binding {
	bindings := layout.KeyMapToSlice(m.viewport.KeyMap)
	return bindings
}

func NewMessagesCmp(app *app.App) tea.Model {
	s := spinner.New()
	s.Spinner = spinner.Pulse
	return &messagesCmp{
		app:           app,
		writingMode:   true,
		cachedContent: make(map[string][]uiMessage),
		viewport:      viewport.New(0, 0),
		spinner:       s,
	}
}
