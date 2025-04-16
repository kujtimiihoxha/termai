package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/kujtimiihoxha/opencode/internal/app"
	"github.com/kujtimiihoxha/opencode/internal/llm/agent"
	"github.com/kujtimiihoxha/opencode/internal/llm/models"
	"github.com/kujtimiihoxha/opencode/internal/llm/tools"
	"github.com/kujtimiihoxha/opencode/internal/logging"
	"github.com/kujtimiihoxha/opencode/internal/message"
	"github.com/kujtimiihoxha/opencode/internal/pubsub"
	"github.com/kujtimiihoxha/opencode/internal/session"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
	"github.com/kujtimiihoxha/opencode/internal/tui/util"
)

type uiMessageType int

const (
	userMessageType uiMessageType = iota
	assistantMessageType
	toolMessageType
)

// messagesTickMsg is a message sent by the timer to refresh messages
type messagesTickMsg time.Time

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
	spinner       spinner.Model
	needsRerender bool
}

func (m *messagesCmp) Init() tea.Cmd {
	return tea.Batch(m.viewport.Init(), m.spinner.Tick, m.tickMessages())
}

func (m *messagesCmp) tickMessages() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return messagesTickMsg(t)
	})
}

func (m *messagesCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case messagesTickMsg:
		// Refresh messages if we have an active session
		if m.session.ID != "" {
			messages, err := m.app.Messages.List(context.Background(), m.session.ID)
			if err == nil {
				m.messages = messages
				m.needsRerender = true
			}
		}
		// Continue ticking
		cmds = append(cmds, m.tickMessages())
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
		m.cachedContent = make(map[string]string)
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
					// If we have messages, ensure the previous last message is not cached
					if len(m.messages) > 0 {
						lastMsgID := m.messages[len(m.messages)-1].ID
						delete(m.cachedContent, lastMsgID)
					}

					m.messages = append(m.messages, msg.Payload)
					delete(m.cachedContent, m.currentMsgID)
					m.currentMsgID = msg.Payload.ID
					m.needsRerender = true
				}
			}
			for _, v := range m.messages {
				for _, c := range v.ToolCalls() {
					if c.ID == msg.Payload.SessionID {
						m.needsRerender = true
					}
				}
			}
		} else if msg.Type == pubsub.UpdatedEvent && msg.Payload.SessionID == m.session.ID {
			logging.Debug("Message", "finish", msg.Payload.FinishReason())
			for i, v := range m.messages {
				if v.ID == msg.Payload.ID {
					m.messages[i] = msg.Payload
					delete(m.cachedContent, msg.Payload.ID)

					// If this is the last message, ensure it's not cached
					if i == len(m.messages)-1 {
						delete(m.cachedContent, msg.Payload.ID)
					}

					m.needsRerender = true
					break
				}
			}
		}
	}

	oldPos := m.viewport.YPosition
	u, cmd := m.viewport.Update(msg)
	m.viewport = u
	m.needsRerender = m.needsRerender || m.viewport.YPosition != oldPos
	cmds = append(cmds, cmd)

	spinner, cmd := m.spinner.Update(msg)
	m.spinner = spinner
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

func (m *messagesCmp) IsAgentWorking() bool {
	return m.app.CoderAgent.IsSessionBusy(m.session.ID)
}

func (m *messagesCmp) renderSimpleMessage(msg message.Message, info ...string) string {
	// Check if this is the last message in the list
	isLastMessage := len(m.messages) > 0 && m.messages[len(m.messages)-1].ID == msg.ID

	// Only use cache for non-last messages
	if !isLastMessage {
		if v, ok := m.cachedContent[msg.ID]; ok {
			return v
		}
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

	// Only cache if it's not the last message
	if !isLastMessage {
		m.cachedContent[msg.ID] = rendered
	}

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

func (m *messagesCmp) findToolResponse(callID string) *message.ToolResult {
	for _, v := range m.messages {
		for _, c := range v.ToolResults() {
			if c.ToolCallID == callID {
				return &c
			}
		}
	}
	return nil
}

func (m *messagesCmp) renderToolCall(toolCall message.ToolCall, isNested bool) string {
	key := ""
	value := ""
	result := styles.BaseStyle.Foreground(styles.PrimaryColor).Render(m.spinner.View() + " waiting for response...")

	response := m.findToolResponse(toolCall.ID)
	if response != nil && response.IsError {
		// Clean up error message for display by removing newlines
		// This ensures error messages display properly in the UI
		errMsg := strings.ReplaceAll(response.Content, "\n", " ")
		result = styles.BaseStyle.Foreground(styles.Error).Render(ansi.Truncate(errMsg, 40, "..."))
	} else if response != nil {
		result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render("Done")
	}
	switch toolCall.Name {
	// TODO: add result data to the tools
	case agent.AgentToolName:
		key = "Task"
		var params agent.AgentParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = strings.ReplaceAll(params.Prompt, "\n", " ")
		if response != nil && !response.IsError {
			firstRow := strings.ReplaceAll(response.Content, "\n", " ")
			result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(ansi.Truncate(firstRow, 40, "..."))
		}
	case tools.BashToolName:
		key = "Bash"
		var params tools.BashParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.Command
		if response != nil && !response.IsError {
			metadata := tools.BashResponseMetadata{}
			json.Unmarshal([]byte(response.Metadata), &metadata)
			result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("Took %s", formatTimeDifference(metadata.StartTime, metadata.EndTime)))
		}

	case tools.EditToolName:
		key = "Edit"
		var params tools.EditParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.FilePath
		if response != nil && !response.IsError {
			metadata := tools.EditResponseMetadata{}
			json.Unmarshal([]byte(response.Metadata), &metadata)
			result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("%d Additions %d Removals", metadata.Additions, metadata.Removals))
		}
	case tools.FetchToolName:
		key = "Fetch"
		var params tools.FetchParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.URL
		if response != nil && !response.IsError {
			result = styles.BaseStyle.Foreground(styles.Error).Render(response.Content)
		}
	case tools.GlobToolName:
		key = "Glob"
		var params tools.GlobParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		if params.Path == "" {
			params.Path = "."
		}
		value = fmt.Sprintf("%s (%s)", params.Pattern, params.Path)
		if response != nil && !response.IsError {
			metadata := tools.GlobResponseMetadata{}
			json.Unmarshal([]byte(response.Metadata), &metadata)
			if metadata.Truncated {
				result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("%d files found (truncated)", metadata.NumberOfFiles))
			} else {
				result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("%d files found", metadata.NumberOfFiles))
			}
		}
	case tools.GrepToolName:
		key = "Grep"
		var params tools.GrepParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		if params.Path == "" {
			params.Path = "."
		}
		value = fmt.Sprintf("%s (%s)", params.Pattern, params.Path)
		if response != nil && !response.IsError {
			metadata := tools.GrepResponseMetadata{}
			json.Unmarshal([]byte(response.Metadata), &metadata)
			if metadata.Truncated {
				result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("%d files found (truncated)", metadata.NumberOfMatches))
			} else {
				result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("%d files found", metadata.NumberOfMatches))
			}
		}
	case tools.LSToolName:
		key = "ls"
		var params tools.LSParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		if params.Path == "" {
			params.Path = "."
		}
		value = params.Path
		if response != nil && !response.IsError {
			metadata := tools.LSResponseMetadata{}
			json.Unmarshal([]byte(response.Metadata), &metadata)
			if metadata.Truncated {
				result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("%d files found (truncated)", metadata.NumberOfFiles))
			} else {
				result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("%d files found", metadata.NumberOfFiles))
			}
		}
	case tools.SourcegraphToolName:
		key = "Sourcegraph"
		var params tools.SourcegraphParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		value = params.Query
		if response != nil && !response.IsError {
			metadata := tools.SourcegraphResponseMetadata{}
			json.Unmarshal([]byte(response.Metadata), &metadata)
			if metadata.Truncated {
				result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("%d matches found (truncated)", metadata.NumberOfMatches))
			} else {
				result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("%d matches found", metadata.NumberOfMatches))
			}
		}
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
		if response != nil && !response.IsError {
			metadata := tools.WriteResponseMetadata{}
			json.Unmarshal([]byte(response.Metadata), &metadata)

			result = styles.BaseStyle.Foreground(styles.ForgroundMid).Render(fmt.Sprintf("%d Additions %d Removals", metadata.Additions, metadata.Removals))
		}
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
			Render(
				ansi.Truncate(
					value+" ",
					m.width-lipgloss.Width(keyValye)-2-lipgloss.Width(result),
					"...",
				),
			)
		value += result

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
		messages, _ := m.app.Messages.List(context.Background(), toolCall.ID)
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

	// If we have messages, ensure the last message is not cached
	// This ensures we always render the latest content for the most recent message
	// which may be actively updating (e.g., during generation)
	if len(m.messages) > 0 {
		lastMsgID := m.messages[len(m.messages)-1].ID
		delete(m.cachedContent, lastMsgID)
	}

	// Limit cache to 10 messages
	if len(m.cachedContent) > 15 {
		// Create a list of keys to delete (oldest messages first)
		keys := make([]string, 0, len(m.cachedContent))
		for k := range m.cachedContent {
			keys = append(keys, k)
		}
		// Delete oldest messages until we have 10 or fewer
		for i := 0; i < len(keys)-15; i++ {
			delete(m.cachedContent, keys[i])
		}
	}

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

	if m.IsAgentWorking() {
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
	messages, err := m.app.Messages.List(context.Background(), session.ID)
	if err != nil {
		return util.ReportError(err)
	}
	m.messages = messages
	m.currentMsgID = m.messages[len(m.messages)-1].ID
	m.needsRerender = true
	m.cachedContent = make(map[string]string)
	return nil
}

func (m *messagesCmp) BindingKeys() []key.Binding {
	bindings := layout.KeyMapToSlice(m.viewport.KeyMap)
	return bindings
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
