package chat

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/message"
	"github.com/kujtimiihoxha/termai/internal/pubsub"
	"github.com/kujtimiihoxha/termai/internal/session"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
	"github.com/kujtimiihoxha/termai/internal/tui/util"
)

type uiMessage struct {
	position int
	height   int
	content  string
}

type messagesCmp struct {
	app           *app.App
	width, height int
	writingMode   bool
	viewport      viewport.Model
	session       session.Session
	messages      []message.Message
	uiMessages    []uiMessage
	currentIndex  int
	renderer      *glamour.TermRenderer
	focusRenderer *glamour.TermRenderer
	cachedContent map[string]string
}

func (m *messagesCmp) Init() tea.Cmd {
	return m.viewport.Init()
}

var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

func hexToBgSGR(hex string) (string, error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return "", fmt.Errorf("invalid hex color: must be 6 hexadecimal digits")
	}

	// Parse RGB components in one block
	rgb := make([]uint64, 3)
	for i := 0; i < 3; i++ {
		val, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
		if err != nil {
			return "", err
		}
		rgb[i] = val
	}

	return fmt.Sprintf("48;2;%d;%d;%d", rgb[0], rgb[1], rgb[2]), nil
}

func forceReplaceBackgroundColors(input string, newBg string) string {
	return ansiEscape.ReplaceAllStringFunc(input, func(seq string) string {
		// Extract content between "\x1b[" and "m"
		content := seq[2 : len(seq)-1]
		tokens := strings.Split(content, ";")
		var newTokens []string

		// Skip background color tokens
		for i := 0; i < len(tokens); i++ {
			if tokens[i] == "" {
				continue
			}

			val, err := strconv.Atoi(tokens[i])
			if err != nil {
				newTokens = append(newTokens, tokens[i])
				continue
			}

			// Skip background color tokens
			if val == 48 {
				// Skip "48;5;N" or "48;2;R;G;B" sequences
				if i+1 < len(tokens) {
					if nextVal, err := strconv.Atoi(tokens[i+1]); err == nil {
						switch nextVal {
						case 5:
							i += 2 // Skip "5" and color index
						case 2:
							i += 4 // Skip "2" and RGB components
						}
					}
				}
			} else if (val < 40 || val > 47) && (val < 100 || val > 107) && val != 49 {
				// Keep non-background tokens
				newTokens = append(newTokens, tokens[i])
			}
		}

		// Add new background if provided
		if newBg != "" {
			newTokens = append(newTokens, strings.Split(newBg, ";")...)
		}

		if len(newTokens) == 0 {
			return ""
		}

		return "\x1b[" + strings.Join(newTokens, ";") + "m"
	})
}

func (m *messagesCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case EditorFocusMsg:
		m.writingMode = bool(msg)
	case SessionSelectedMsg:
		if msg.ID != m.session.ID {
			cmd := m.SetSession(msg)
			return m, cmd
		}
		return m, nil
	case pubsub.Event[message.Message]:
		if msg.Type == pubsub.CreatedEvent {
			if msg.Payload.SessionID == m.session.ID {
				// check if message exists
				for _, v := range m.messages {
					if v.ID == msg.Payload.ID {
						return m, nil
					}
				}

				m.messages = append(m.messages, msg.Payload)
				m.renderView()
				m.viewport.GotoBottom()
			}
			for _, v := range m.messages {
				for _, c := range v.ToolCalls() {
					// the message is being added to the session of a tool called
					if c.ID == msg.Payload.SessionID {
						m.renderView()
						m.viewport.GotoBottom()
					}
				}
			}
		} else if msg.Type == pubsub.UpdatedEvent && msg.Payload.SessionID == m.session.ID {
			for i, v := range m.messages {
				if v.ID == msg.Payload.ID {
					m.messages[i] = msg.Payload
					delete(m.cachedContent, msg.Payload.ID)
					m.renderView()
					if i == len(m.messages)-1 {
						m.viewport.GotoBottom()
					}
					break
				}
			}
		}
	}
	u, cmd := m.viewport.Update(msg)
	m.viewport = u
	return m, cmd
}

func (m *messagesCmp) renderUserMessage(inx int, msg message.Message) string {
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
	if inx == m.currentIndex {
		style = style.
			Foreground(styles.Forground).
			BorderForeground(styles.Blue).
			BorderStyle(lipgloss.ThickBorder())
		renderer = m.focusRenderer
	}
	c, _ := renderer.Render(msg.Content().String())
	col, _ := hexToBgSGR(styles.Background.Dark)
	rendered := style.Render(forceReplaceBackgroundColors(c, col))
	m.cachedContent[msg.ID] = rendered
	return rendered
}

func (m *messagesCmp) renderView() {
	m.uiMessages = make([]uiMessage, 0)
	pos := 0

	for _, v := range m.messages {
		content := ""
		switch v.Role {
		case message.User:
			content = m.renderUserMessage(pos, v)
		}
		m.uiMessages = append(m.uiMessages, uiMessage{
			position: pos,
			height:   lipgloss.Height(content),
			content:  content,
		})
		pos += lipgloss.Height(content) + 1 // + 1 for spacing
	}

	messages := make([]string, 0)
	for _, v := range m.uiMessages {
		messages = append(messages, v.content)
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

	m.renderView()
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
	if m.writingMode {
		text = lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.BaseStyle.Foreground(styles.ForgroundDim).Bold(true).Render("press "),
			styles.BaseStyle.Foreground(styles.Forground).Bold(true).Render("esc"),
			styles.BaseStyle.Foreground(styles.ForgroundDim).Bold(true).Render(" to exit writing mode"),
		)
	} else {
		text = lipgloss.JoinHorizontal(
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
	m.renderer = renderer
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
	m.messages = append(m.messages, m.messages[0])
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
	return &messagesCmp{
		app:           app,
		writingMode:   true,
		cachedContent: make(map[string]string),
		viewport:      viewport.New(0, 0),
		focusRenderer: focusRenderer,
		renderer:      renderer,
	}
}
