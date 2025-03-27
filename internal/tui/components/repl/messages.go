package repl

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/message"
	"github.com/kujtimiihoxha/termai/internal/pubsub"
	"github.com/kujtimiihoxha/termai/internal/session"
	"github.com/kujtimiihoxha/termai/internal/tui/layout"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
)

type MessagesCmp interface {
	tea.Model
	layout.Focusable
	layout.Bordered
	layout.Sizeable
	layout.Bindings
}

type messagesCmp struct {
	app            *app.App
	messages       []message.Message
	selectedMsgIdx int // Index of the selected message
	session        session.Session
	viewport       viewport.Model
	mdRenderer     *glamour.TermRenderer
	width          int
	height         int
	focused        bool
	cachedView     string
}

func (m *messagesCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pubsub.Event[message.Message]:
		if msg.Type == pubsub.CreatedEvent {
			m.messages = append(m.messages, msg.Payload)
			m.renderView()
			m.viewport.GotoBottom()
		} else if msg.Type == pubsub.UpdatedEvent {
			for i, v := range m.messages {
				if v.ID == msg.Payload.ID {
					m.messages[i] = msg.Payload
					m.renderView()
					if i == len(m.messages)-1 {
						m.viewport.GotoBottom()
					}
					break
				}
			}
		}
	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent {
			if m.session.ID == msg.Payload.ID {
				m.session = msg.Payload
			}
		}
	case SelectedSessionMsg:
		m.session, _ = m.app.Sessions.Get(msg.SessionID)
		m.messages, _ = m.app.Messages.List(m.session.ID)
		m.renderView()
		m.viewport.GotoBottom()
	}
	if m.focused {
		u, cmd := m.viewport.Update(msg)
		m.viewport = u
		return m, cmd
	}
	return m, nil
}

func borderColor(role message.MessageRole) lipgloss.TerminalColor {
	switch role {
	case message.Assistant:
		return styles.Mauve
	case message.User:
		return styles.Rosewater
	}
	return styles.Blue
}

func borderText(msgRole message.MessageRole, currentMessage int) map[layout.BorderPosition]string {
	role := ""
	icon := ""
	switch msgRole {
	case message.Assistant:
		role = "Assistant"
		icon = styles.BotIcon
	case message.User:
		role = "User"
		icon = styles.UserIcon
	}
	return map[layout.BorderPosition]string{
		layout.TopLeftBorder: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(styles.Crust).
			Background(borderColor(msgRole)).
			Render(fmt.Sprintf("%s %s ", role, icon)),
		layout.TopRightBorder: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(styles.Crust).
			Background(borderColor(msgRole)).
			Render(fmt.Sprintf("#%d ", currentMessage)),
	}
}

func renderMessageWithToolCall(content string, tools []message.ToolCall, futureMessages []message.Message) string {
	// Container for the message and tool calls
	allParts := []string{content}

	// Connector style
	connectorStyle := lipgloss.NewStyle().
		Foreground(styles.Peach).
		Bold(true)

	// Tool call container style
	toolCallStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Peach).
		Padding(0, 1)

	// Tool result style
	toolResultStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Green).
		Padding(0, 1)

	// Running indicator style
	runningStyle := lipgloss.NewStyle().
		Foreground(styles.Peach).
		Bold(true)

	renderTool := func(toolCall message.ToolCall) string {
		toolHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.Blue).
			Render(fmt.Sprintf("%s %s", styles.ToolIcon, toolCall.Name))

		var paramLines []string
		var args map[string]interface{}
		var paramOrder []string

		// Unmarshal JSON into the map
		json.Unmarshal([]byte(toolCall.Input), &args)

		for key := range args {
			paramOrder = append(paramOrder, key)
		}
		sort.Strings(paramOrder)

		// Now iterate through parameters in their original order
		for _, name := range paramOrder {
			value := args[name]
			paramName := lipgloss.NewStyle().
				Foreground(styles.Peach).
				Bold(true).
				Render(name)

			truncate := 50
			if len(fmt.Sprintf("%v", value)) > truncate {
				value = fmt.Sprintf("%v", value)[:truncate] + lipgloss.NewStyle().Foreground(styles.Blue).Render("... (truncated)")
			}
			paramValue := fmt.Sprintf("%v", value)
			paramLines = append(paramLines, fmt.Sprintf("  %s: %s", paramName, paramValue))
		}

		paramBlock := lipgloss.JoinVertical(lipgloss.Left, paramLines...)

		toolContent := lipgloss.JoinVertical(lipgloss.Left, toolHeader, paramBlock)
		return toolCallStyle.Render(toolContent)
	}

	// Find tool results for each tool call
	findToolResult := func(toolCallID string, messages []message.Message) *message.ToolResult {
		for _, msg := range messages {
			if msg.Role == message.Tool {
				for _, result := range msg.ToolResults {
					if result.ToolCallID == toolCallID {
						return &result
					}
				}
			}
		}
		return nil
	}

	renderToolResult := func(result message.ToolResult) string {
		resultHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.Green).
			Render(fmt.Sprintf("%s Result", styles.CheckIcon))

		// Truncate long outputs
		truncate := 200
		content := result.Content
		if len(content) > truncate {
			content = content[:truncate] + lipgloss.NewStyle().Foreground(styles.Blue).Render("... (truncated)")
		}

		resultContent := lipgloss.JoinVertical(lipgloss.Left, resultHeader, content)
		return toolResultStyle.Render(resultContent)
	}

	// Add connector after original content
	connector := connectorStyle.Render("└─> Tool Calls:")
	allParts = append(allParts, connector)

	// Add all tool calls with their results if available
	for _, toolCall := range tools {
		toolOutput := renderTool(toolCall)
		allParts = append(allParts, "    "+strings.ReplaceAll(toolOutput, "\n", "\n    "))

		// Check if we have a result for this tool call
		result := findToolResult(toolCall.ID, futureMessages)
		if result != nil {
			resultOutput := renderToolResult(*result)
			allParts = append(allParts, "    "+strings.ReplaceAll(resultOutput, "\n", "\n    "))
		} else {
			// Show running indicator if no result yet
			runningIndicator := runningStyle.Render(fmt.Sprintf("%s Running...", styles.SpinnerIcon))
			allParts = append(allParts, "    "+runningIndicator)
		}
	}

	// Add tool calls from future messages
	for _, msg := range futureMessages {
		if msg.Content != "" {
			break
		}

		for _, toolCall := range msg.ToolCalls {
			toolOutput := renderTool(toolCall)
			allParts = append(allParts, "    "+strings.ReplaceAll(toolOutput, "\n", "\n    "))

			// Check if we have a result for this tool call
			result := findToolResult(toolCall.ID, futureMessages)
			if result != nil {
				resultOutput := renderToolResult(*result)
				allParts = append(allParts, "    "+strings.ReplaceAll(resultOutput, "\n", "\n    "))
			} else {
				// Show running indicator if no result yet
				runningIndicator := runningStyle.Render(fmt.Sprintf("%s Running...", styles.SpinnerIcon))
				allParts = append(allParts, "    "+runningIndicator)
			}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, allParts...)
}

func (m *messagesCmp) renderView() {
	stringMessages := make([]string, 0)
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(styles.CatppuccinMarkdownStyle()),
		glamour.WithWordWrap(m.width-10),
		glamour.WithEmoji(),
	)
	textStyle := lipgloss.NewStyle().Width(m.width - 4)
	currentMessage := 1
	displayedMsgCount := 0 // Track the actual displayed messages count

	// find all messages that have content
	prevMessageWasUser := false
	for inx, msg := range m.messages {
		content := msg.Content
		if content != "" || prevMessageWasUser {
			if content == "" {
				content = "..."
			}
			content, _ = r.Render(content)

			// Check if this message is the selected one
			isSelected := inx == m.selectedMsgIdx

			// Determine border style based on selection
			border := lipgloss.DoubleBorder()
			activeColor := borderColor(msg.Role)

			// Highlight the selected message with a brighter border
			if isSelected {
				activeColor = styles.Primary // Use primary color for selected message
			}

			content = layout.Borderize(
				textStyle.Render(content),
				layout.BorderOptions{
					InactiveBorder: border,
					ActiveBorder:   border,
					ActiveColor:    activeColor,
					InactiveColor:  borderColor(msg.Role),
					EmbeddedText:   borderText(msg.Role, currentMessage),
				},
			)
			if len(msg.ToolCalls) > 0 {
				content = renderMessageWithToolCall(content, msg.ToolCalls, m.messages[inx+1:])
			}
			stringMessages = append(stringMessages, content)
			currentMessage++
			displayedMsgCount++
		}
		if msg.Role == message.User && msg.Content != "" {
			prevMessageWasUser = true
		} else {
			prevMessageWasUser = false
		}
	}
	m.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Top, stringMessages...))
}

func (m *messagesCmp) View() string {
	return lipgloss.NewStyle().Padding(1).Render(m.viewport.View())
}

func (m *messagesCmp) BindingKeys() []key.Binding {
	keys := layout.KeyMapToSlice(m.viewport.KeyMap)

	// Add message selection keybindings
	selectionKeys := []key.Binding{
		key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "next message"),
		),
		key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "previous message"),
		),
	}

	return append(keys, selectionKeys...)
}

func (m *messagesCmp) Blur() tea.Cmd {
	m.focused = false
	return nil
}

func (m *messagesCmp) BorderText() map[layout.BorderPosition]string {
	title := m.session.Title
	titleWidth := m.width / 2
	if len(title) > titleWidth {
		title = title[:titleWidth] + "..."
	}
	if m.focused {
		title = lipgloss.NewStyle().Foreground(styles.Primary).Render(title)
	}
	return map[layout.BorderPosition]string{
		layout.TopLeftBorder:     title,
		layout.BottomRightBorder: formatTokensAndCost(m.session.CompletionTokens+m.session.PromptTokens, m.session.Cost),
	}
}

func (m *messagesCmp) Focus() tea.Cmd {
	m.focused = true
	return nil
}

func (m *messagesCmp) GetSize() (int, int) {
	return m.width, m.height
}

func (m *messagesCmp) IsFocused() bool {
	return m.focused
}

func (m *messagesCmp) SetSize(width int, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width - 2   // padding
	m.viewport.Height = height - 2 // padding
	m.renderView()
}

func (m *messagesCmp) Init() tea.Cmd {
	return nil
}

func NewMessagesCmp(app *app.App) MessagesCmp {
	return &messagesCmp{
		app:      app,
		messages: []message.Message{},
		viewport: viewport.New(0, 0),
	}
}
